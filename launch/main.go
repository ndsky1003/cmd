package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
)

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
	flag.Parse()
}

func main() {
	if sub_exe == "" {
		fmt.Println("子进程不能为空")
		return
	}
	if err := os.Mkdir(logger.BackDir, 0755); err != nil && !os.IsExist(err) {
		panic(err)
	}
	log.SetOutput(logger)
	defer logger.Close()
	launch()
}

func launch() {
	exepath, err := exec.LookPath(sub_exe)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	args := os.Args[1:]
	newArgs := make([]string, 0, len(args))
	var isSkip bool
	for i := 0; i < len(args); i++ {
		if isSkip {
			isSkip = false
			continue
		}
		if a := args[i]; a == "-r" {
			isSkip = true
			continue
		}
		newArgs = append(newArgs, args[i])
	}

	cmd := exec.Command(exepath, newArgs...)
	cmd.Stdout = logger
	cmd.Stderr = logger
	err = cmd.Run()
	fmt.Println("process 正常 exit!", "exepath:", exepath, "err:", err, "args:", args, "newArgs:", newArgs)
}
