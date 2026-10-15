package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setMockTime 用可变的 now 函数替换 currentTime，返回恢复函数。
func setMockTime(now func() time.Time) func() {
	old := currentTime
	currentTime = now
	return func() { currentTime = old }
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

// countSubstr 统计目录下所有文件内容里 substr 出现的次数。
func countSubstr(t *testing.T, dir, substr string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read file %s: %v", e.Name(), err)
		}
		count += strings.Count(string(b), substr)
	}
	return count
}
