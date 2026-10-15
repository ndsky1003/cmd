package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ndsky1003/cmd/common/version"
	"github.com/ndsky1003/cmd/launch/logger"
)

var Version = "dev"

var sub_exe string

// 日志配置，默认值由 init() 中 flag 注册填充。
var (
	cfgBackDir    string
	cfgFilename   string
	cfgMaxSize    int
	cfgMaxBackups int
	cfgMaxAge     int
	cfgCompress   bool
	cfgLocalTime  bool
	cfgSplit      string
)

func init() {
	flag.StringVar(&sub_exe, "r", "", "sub exe（可选，也可直接作为第一个位置参数指定）")
	flag.StringVar(&cfgBackDir, "dir", "log", "log dir")
	flag.StringVar(&cfgFilename, "filename", "main.log", "log name")
	flag.IntVar(&cfgMaxSize, "maxsize", 100, "max size (M)")
	flag.IntVar(&cfgMaxBackups, "maxbackups", 30, "max backups (数量)")
	flag.IntVar(&cfgMaxAge, "maxage", 28, "max age (大小模式:天 / 时间模式:周期数)")
	flag.BoolVar(&cfgCompress, "compress", false, "true compress,false no compress")
	flag.BoolVar(&cfgLocalTime, "localtime", true, "true use localtime,false use utc")
	flag.StringVar(&cfgSplit, "split", "size", "log split: size|hour|day|week|<duration>，或 size+day 等混合")
	v := flag.Bool("v", false, "print version information and exit")
	flag.BoolVar(v, "version", false, "same as -v")
	flag.Parse()
	if *v {
		fmt.Printf("launch version %s\n", version.GetVersion(Version))
		os.Exit(0)
	}
}

// parseSplit 解析 -split 参数，返回是否含大小轮转、时间周期及错误。
// 例："size"、"day"、"hour"、"6h"、"size+day"、"size+6h"。
func parseSplit(s string) (hasSize bool, interval time.Duration, err error) {
	if s == "" {
		s = "size"
	}
	for _, part := range strings.Split(s, "+") {
		switch part {
		case "size":
			hasSize = true
		case "hour":
			interval = time.Hour
		case "day":
			interval = 24 * time.Hour
		case "week":
			interval = 7 * 24 * time.Hour
		default:
			d, e := time.ParseDuration(part)
			if e != nil {
				return false, 0, fmt.Errorf("unknown split mode: %q", part)
			}
			interval = d
		}
	}
	return hasSize, interval, nil
}

// buildLogger 根据 -split 构建 logger：纯大小 / 纯时间间隔 / size+time 混合。
func buildLogger() (io.WriteCloser, error) {
	hasSize, interval, err := parseSplit(cfgSplit)
	if err != nil {
		return nil, err
	}

	switch {
	case hasSize && interval > 0:
		// size + time 混合：大小与时间边界先到先触发，备份用时间戳命名。
		return &logger.SizeLogger{
			BackDir:    cfgBackDir,
			Filename:   cfgFilename,
			MaxSize:    cfgMaxSize,
			MaxBackups: cfgMaxBackups,
			MaxAge:     cfgMaxAge,
			Compress:   cfgCompress,
			LocalTime:  cfgLocalTime,
			Interval:   interval,
		}, nil
	case interval > 0:
		// 纯时间间隔分割。
		return &logger.IntervalLogger{
			BackDir:    cfgBackDir,
			Filename:   cfgFilename,
			Interval:   interval,
			MaxAge:     cfgMaxAge,
			MaxBackups: cfgMaxBackups,
			Compress:   cfgCompress,
			LocalTime:  cfgLocalTime,
		}, nil
	default:
		// 纯大小分割。
		return &logger.SizeLogger{
			BackDir:    cfgBackDir,
			Filename:   cfgFilename,
			MaxSize:    cfgMaxSize,
			MaxBackups: cfgMaxBackups,
			MaxAge:     cfgMaxAge,
			Compress:   cfgCompress,
			LocalTime:  cfgLocalTime,
		}, nil
	}
}

func main() {
	// 解析子进程及其参数：
	// 1. 优先使用 -r 指定的 exe，此时 flag.Args() 全部作为子进程参数。
	// 2. 未指定 -r 时，取第一个位置参数作为 exe，其余作为子进程参数。
	subArgs := flag.Args()
	if sub_exe == "" {
		if len(subArgs) == 0 {
			slog.Info("子进程不能为空")
			return
		}
		sub_exe = subArgs[0]
		subArgs = subArgs[1:]
	}

	logWriter, err := buildLogger()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(2)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(logWriter, nil)))
	defer logWriter.Close()
	launch(subArgs, logWriter)
}

func launch(args []string, logWriter io.WriteCloser) {
	exepath, err := exec.LookPath(sub_exe)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	// 若子进程参数以 `-` 开头，需用 `--` 分隔，
	// 例如：launch prog -- --foo bar
	cmd := exec.Command(exepath, args...)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	err = cmd.Run()
	slog.Info("process 正常 exit!", "exepath:", exepath, "err:", err, "args:", args)
}
