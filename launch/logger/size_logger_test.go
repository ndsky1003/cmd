package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestSizeLoggerRotate(t *testing.T) {
	dir := t.TempDir()
	l := &SizeLogger{Filename: "app.log", BackDir: dir, MaxSize: 1}
	defer l.Close()

	// 每次写 1KB，累计超过 1MB 触发一次轮转。
	chunk := bytes.Repeat([]byte("x"), 1024)
	for i := 0; i < 1100; i++ {
		if _, err := l.Write(chunk); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 2 {
		t.Fatalf("expected current + backup files after rotation, got %d", len(entries))
	}
	hasBackup := false
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "app-") {
			hasBackup = true
		}
	}
	if !hasBackup {
		t.Fatalf("expected a backup file with prefix app-, got files: %v", entryNames(entries))
	}
}

func TestSizeLoggerMaxBackups(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "app-2026-10-01T00-00-00.000.log"), "1")
	writeFile(t, filepath.Join(dir, "app-2026-10-02T00-00-00.000.log"), "2")
	writeFile(t, filepath.Join(dir, "app-2026-10-03T00-00-00.000.log"), "3")

	l := &SizeLogger{Filename: "app.log", BackDir: dir, MaxBackups: 2}
	if err := l.millRunOnce(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-01T00-00-00.000.log")); !os.IsNotExist(err) {
		t.Fatalf("expected oldest backup removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-03T00-00-00.000.log")); err != nil {
		t.Fatalf("expected newest backup kept: %v", err)
	}
}

func TestSizeLoggerCompress(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "app-2026-10-01T00-00-00.000.log"), "hello compress")

	l := &SizeLogger{Filename: "app.log", BackDir: dir, Compress: true}
	if err := l.millRunOnce(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-01T00-00-00.000.log")); !os.IsNotExist(err) {
		t.Fatalf("expected original removed after compress, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-01T00-00-00.000.log.gz")); err != nil {
		t.Fatalf("expected .gz file: %v", err)
	}
}

func TestConcurrentWrite(t *testing.T) {
	dir := t.TempDir()
	l := &SizeLogger{Filename: "app.log", BackDir: dir, MaxSize: 100}

	const goroutines = 8
	const writes = 1000
	line := []byte("hello world\n")

	var wg sync.WaitGroup
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < writes; i++ {
				if _, err := l.Write(line); err != nil {
					t.Errorf("write: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}

	if got := countSubstr(t, dir, "hello world"); got != goroutines*writes {
		t.Fatalf("expected %d lines, got %d", goroutines*writes, got)
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}
