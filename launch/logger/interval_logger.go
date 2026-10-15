package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// reportError 清理/压缩失败的上报：优先回调，否则标准库 log 输出。
func reportError(onError func(error), err error) {
	if err == nil {
		return
	}
	if onError != nil {
		onError(err)
		return
	}
	log.Printf("logger: %v", err)
}

// periodStart 返回 t 所在周期的起始时刻。
// hour/day/week 使用日历对齐（本地整点/零点/周一零点），
// 其他固定时长（如 6h、30m）以本地零点为基准截断。
func periodStart(t time.Time, interval time.Duration) time.Time {
	y, m, d := t.Date()
	switch interval {
	case time.Hour:
		return time.Date(y, m, d, t.Hour(), 0, 0, 0, t.Location())
	case 24 * time.Hour:
		return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	case 7 * 24 * time.Hour:
		day := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		daysSinceMon := (int(t.Weekday()) + 6) % 7 // 周一 = 0
		return day.AddDate(0, 0, -daysSinceMon)
	default:
		dayStart := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		elapsed := t.Sub(dayStart)
		return dayStart.Add(elapsed.Truncate(interval))
	}
}

// stampLayout 返回文件名中周期标识的格式化布局。
func stampLayout(interval time.Duration) string {
	switch interval {
	case time.Hour:
		return "2006-01-02-15"
	case 24 * time.Hour, 7 * 24 * time.Hour:
		return "2006-01-02"
	default:
		return "2006-01-02T15-04-05"
	}
}

// IntervalLogger 按时间间隔分割日志：每个周期生成一个新文件，
// 文件名形如 main-2006-01-02.log（天/周）、main-2006-01-02-15.log（小时）。
type IntervalLogger struct {
	BackDir    string
	Filename   string
	Interval   time.Duration // 分割周期，如 time.Hour、24*time.Hour、7*24*time.Hour
	MaxAge     int           // 保留最近 N 个周期，0 表示不限
	MaxBackups int           // 保留最大文件数，0 表示不限
	Compress   bool          // 是否压缩旧日志
	LocalTime  bool          // 周期使用本地时间（否则 UTC）

	// OnError 可选：清理/压缩失败时的回调。未设置时使用标准库 log 输出。
	OnError func(error)

	file      *os.File
	mu        sync.Mutex
	stamp     string    // 当前文件对应周期的标识
	nextCheck time.Time // 下次周期边界

	millCh    chan string
	startMill sync.Once
}

var _ io.WriteCloser = (*IntervalLogger)(nil)

func (l *IntervalLogger) Write(p []byte) (n int, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err = l.ensureOpen(); err != nil {
		return 0, err
	}
	n, err = l.file.Write(p)
	return n, err
}

func (l *IntervalLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.close()
}

func (l *IntervalLogger) close() error {
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	l.stamp = ""
	l.nextCheck = time.Time{}
	return err
}

// ensureOpen 确保当前周期对应的文件已打开；跨周期时切换文件并清理旧日志。
// 快速路径只做一次时间比较，不做字符串格式化。
func (l *IntervalLogger) ensureOpen() error {
	now := currentTime()
	if !l.LocalTime {
		now = now.UTC()
	}

	// 仍在当前周期：直接复用已打开的文件。
	if l.file != nil && now.Before(l.nextCheck) {
		return nil
	}

	// 跨周期或首次打开：计算周期起始与下一个边界，仅此分支做日期格式化。
	start := periodStart(now, l.Interval)
	stamp := start.Format(stampLayout(l.Interval))
	next := start.Add(l.Interval)

	if err := l.close(); err != nil {
		return err
	}
	if err := os.MkdirAll(l.dir(), 0755); err != nil {
		return fmt.Errorf("can't make directories for interval logfile: %s", err)
	}
	f, err := os.OpenFile(l.filename(stamp), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("can't open interval logfile: %s", err)
	}
	l.file = f
	l.stamp = stamp
	l.nextCheck = next

	// 异步清理，避免阻塞写路径。
	l.mill(stamp)
	return nil
}

func (l *IntervalLogger) dir() string {
	if l.BackDir != "" {
		return l.BackDir
	}
	return filepath.Dir(l.baseName())
}

func (l *IntervalLogger) baseName() string {
	if l.Filename != "" {
		return l.Filename
	}
	return filepath.Base(os.Args[0]) + "-lumberjack.log"
}

func (l *IntervalLogger) filename(stamp string) string {
	return filepath.Join(l.dir(), l.named(stamp))
}

// named 生成指定周期对应的文件名，如 main-2006-01-02.log。
func (l *IntervalLogger) named(stamp string) string {
	filename := filepath.Base(l.baseName())
	ext := filepath.Ext(filename)
	prefix := filename[:len(filename)-len(ext)]
	return prefix + "-" + stamp + ext
}

// prefixAndExt 返回用于匹配日志文件的前缀与扩展名，如 ("main-", ".log")。
func (l *IntervalLogger) prefixAndExt() (prefix, ext string) {
	filename := filepath.Base(l.baseName())
	ext = filepath.Ext(filename)
	prefix = filename[:len(filename)-len(ext)] + "-"
	return prefix, ext
}

// mill 异步触发清理，清理失败通过 OnError 上报。
func (l *IntervalLogger) mill(stamp string) {
	l.startMill.Do(func() {
		l.millCh = make(chan string, 1)
		go func() {
			for s := range l.millCh {
				reportError(l.OnError, l.cleanOnce(s))
			}
		}()
	})
	select {
	case l.millCh <- stamp:
	default:
	}
}

// cleanOnce 清理过期日志并压缩旧文件，排除当前周期文件。
func (l *IntervalLogger) cleanOnce(curStamp string) error {
	if l.MaxAge == 0 && l.MaxBackups == 0 && !l.Compress {
		return nil
	}
	files, err := l.oldLogFiles(curStamp)
	if err != nil {
		return err
	}

	var remove, compress []intervalLogInfo

	if l.MaxBackups > 0 && l.MaxBackups < len(files) {
		preserved := make(map[string]bool)
		var remaining []intervalLogInfo
		for _, f := range files {
			fn := strings.TrimSuffix(f.Name(), compressSuffix)
			preserved[fn] = true
			if len(preserved) > l.MaxBackups {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}

	if l.MaxAge > 0 {
		curStart, _ := time.Parse(stampLayout(l.Interval), curStamp)
		cutoff := curStart.Add(-time.Duration(l.MaxAge) * l.Interval)
		var remaining []intervalLogInfo
		for _, f := range files {
			if f.stamp.Before(cutoff) {
				remove = append(remove, f)
			} else {
				remaining = append(remaining, f)
			}
		}
		files = remaining
	}

	if l.Compress {
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), compressSuffix) {
				compress = append(compress, f)
			}
		}
	}

	var firstErr error
	for _, f := range remove {
		if e := os.Remove(filepath.Join(l.dir(), f.Name())); e != nil && firstErr == nil {
			firstErr = e
		}
	}
	for _, f := range compress {
		src := filepath.Join(l.dir(), f.Name())
		if e := compressLogFile(src, src+compressSuffix); e != nil && firstErr == nil {
			firstErr = e
		}
	}
	return firstErr
}

// oldLogFiles 返回日志目录下匹配当前前缀的旧日志文件（排除当前周期），按周期倒序。
func (l *IntervalLogger) oldLogFiles(curStamp string) ([]intervalLogInfo, error) {
	entries, err := os.ReadDir(l.dir())
	if err != nil {
		return nil, fmt.Errorf("can't read log file directory: %s", err)
	}
	prefix, ext := l.prefixAndExt()
	nowName := l.named(curStamp)

	var files []intervalLogInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == nowName {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if t, ok := l.stampFromName(name, prefix, ext); ok {
			files = append(files, intervalLogInfo{stamp: t, FileInfo: info})
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].stamp.After(files[j].stamp)
	})
	return files, nil
}

// stampFromName 从文件名中解析周期起始时间，如 main-2026-10-15.log -> 2026-10-15。
func (l *IntervalLogger) stampFromName(name, prefix, ext string) (time.Time, bool) {
	if !strings.HasPrefix(name, prefix) {
		return time.Time{}, false
	}
	s := strings.TrimPrefix(name, prefix)
	// 先匹配压缩后缀（.log.gz），再匹配普通后缀（.log）。
	for _, suffix := range []string{ext + compressSuffix, ext} {
		if strings.HasSuffix(s, suffix) {
			ds := strings.TrimSuffix(s, suffix)
			t, err := time.Parse(stampLayout(l.Interval), ds)
			if err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// intervalLogInfo 包装日志文件信息及其周期起始时间。
type intervalLogInfo struct {
	stamp time.Time
	os.FileInfo
}
