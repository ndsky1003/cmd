package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIntervalLoggerSwitch(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	restore := setMockTime(func() time.Time { return now })
	defer restore()

	l := &IntervalLogger{Filename: "app.log", BackDir: dir, Interval: 24 * time.Hour}
	defer l.Close()

	if _, err := l.Write([]byte("day1\n")); err != nil {
		t.Fatal(err)
	}
	now = now.Add(24 * time.Hour) // 跨天
	if _, err := l.Write([]byte("day2\n")); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-15.log")); err != nil {
		t.Fatalf("expected day1 file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-16.log")); err != nil {
		t.Fatalf("expected day2 file: %v", err)
	}
}

func TestIntervalLoggerMaxAge(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "app-2026-10-10.log"), "old")
	writeFile(t, filepath.Join(dir, "app-2026-10-13.log"), "new")

	l := &IntervalLogger{Filename: "app.log", BackDir: dir, Interval: 24 * time.Hour, MaxAge: 3}
	if err := l.cleanOnce("2026-10-15"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-10.log")); !os.IsNotExist(err) {
		t.Fatalf("expected old file removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-13.log")); err != nil {
		t.Fatalf("expected recent file kept: %v", err)
	}
}

func TestIntervalLoggerMaxBackups(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "app-2026-10-12.log"), "1")
	writeFile(t, filepath.Join(dir, "app-2026-10-13.log"), "2")
	writeFile(t, filepath.Join(dir, "app-2026-10-14.log"), "3")

	l := &IntervalLogger{Filename: "app.log", BackDir: dir, Interval: 24 * time.Hour, MaxBackups: 2}
	if err := l.cleanOnce("2026-10-15"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-12.log")); !os.IsNotExist(err) {
		t.Fatalf("expected oldest removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-14.log")); err != nil {
		t.Fatalf("expected newest kept: %v", err)
	}
}

func TestIntervalLoggerCompress(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "app-2026-10-14.log"), "hello compress")

	l := &IntervalLogger{Filename: "app.log", BackDir: dir, Interval: 24 * time.Hour, Compress: true}
	if err := l.cleanOnce("2026-10-15"); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-14.log.gz")); err != nil {
		t.Fatalf("expected .gz file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app-2026-10-14.log")); !os.IsNotExist(err) {
		t.Fatalf("expected original removed after compress, stat err=%v", err)
	}
}
