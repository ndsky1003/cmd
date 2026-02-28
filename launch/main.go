package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

var Version = "1.0.0"

var logger = &Logger{
	BackDir:    "log",
	Filename:   "main.log",
	MaxSize:    100, // megabytes
	MaxBackups: 30,
	MaxAge:     20, // days
	Compress:   true,
	LocalTime:  true,
}
var sub_exe string

func init() {
	flag.StringVar(&sub_exe, "r", "", "sub exe")
	flag.StringVar(&logger.BackDir, "dir", "log", "log dir")
	flag.StringVar(&logger.Filename, "filename", "main.log", "log name")
	flag.IntVar(&logger.MaxSize, "maxsize", 100, "max size (M)")
	flag.IntVar(&logger.MaxBackups, "maxbackups", 30, "max backups (数量)")
	flag.IntVar(&logger.MaxAge, "maxage", 28, "max age (天)")
	flag.BoolVar(&logger.Compress, "compress", false, "true compress,false no compress")
	flag.BoolVar(&logger.LocalTime, "localtime", true, "true use localtime,false use utc")
	v := flag.Bool("v", false, "print version information and exit")
	flag.BoolVar(v, "version", false, "same as -v")
	flag.Parse()
	if *v {
		fmt.Println(Version)
		os.Exit(0)
	}
}

func main() {
	if sub_exe == "" {
		slog.Info("子进程不能为空")
		return
	}
	if err := os.Mkdir(logger.BackDir, 0755); err != nil && !os.IsExist(err) {
		panic(err)
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(logger, nil)))
	defer logger.Close()
	launch()
}

func launch() {
	exepath, err := exec.LookPath(sub_exe)
	if err != nil {
		slog.Error(err.Error())
		return
	}

	args := os.Args[1:]
	newArgs := make([]string, 0, len(args))

	for i := 0; i < len(args); {
		if args[i] == "-r" || args[i] == "--r" {
			i += 2 // 跳过 -r 和它的值
			continue
		}
		newArgs = append(newArgs, args[i])
		i++
	}

	cmd := exec.Command(exepath, newArgs...)
	cmd.Stdout = logger
	cmd.Stderr = logger
	err = cmd.Run()
	slog.Info("process 正常 exit!", "exepath:", exepath, "err:", err, "args:", args, "newArgs:", newArgs)
}
