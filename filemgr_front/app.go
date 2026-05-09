package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/ndsky1003/crpc/v3"
	"github.com/ndsky1003/crpc/v3/protocol"
)

type App struct {
	ctx     context.Context
	client  *crpc.Client
	target  string // filemgr 服务端注册的服务名
	module  string // filemgr 注册的模块名（RegisterName）
}

type FileInfo struct {
	Name  string
	IsDir bool
	Ext   string
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) Connect(addr, name, secret string) error {
	if name == "" {
		name = "filemgr_front"
	}
	if addr == "" {
		return fmt.Errorf("address is required")
	}
	// 默认行为：目标服务名和模块名都取 name
	// 若 name 含斜杠如 "filemgr/crpc"，则拆分为 target/module
	target, module := name, "crpc"
	for i, c := range name {
		if c == '/' {
			target = name[:i]
			module = name[i+1:]
			break
		}
	}

	client, err := crpc.Dial(context.Background(), "filemgr_front", addr, crpc.ClientOptions().SetSecret(secret))
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	a.client = client
	a.target = target
	a.module = module
	return nil
}

func (a *App) Disconnect() error {
	if a.client != nil {
		if err := a.client.Close(); err != nil {
			return err
		}
		a.client = nil
	}
	return nil
}

func (a *App) call(method string, args, reply any) error {
	return a.client.Call(context.Background(), a.target, a.module+"."+method, args, reply)
}

func (a *App) ListDir(path string) ([]*FileInfo, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	var files []*FileInfo
	err := a.call("ListDir", struct{ Path string }{Path: path}, &files)
	return files, err
}

func (a *App) Mkdir(path string) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}
	return a.call("Mkdir", struct{ Path string }{Path: path}, nil)
}

func (a *App) SaveFile(filename string, data []byte) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}
	req := &protocol.FileTransfer{
		FileName: filename,
		Data:     data,
		Offset:   0,
		IsFinish: true,
	}
	return a.call("SaveFile", req, nil)
}

func (a *App) ReadFile(path string) ([]byte, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	var data []byte
	err := a.call("ReadFile", struct{ Path string }{Path: path}, &data)
	return data, err
}

func (a *App) OpenFile(path string) error {
	data, err := a.ReadFile(path)
	if err != nil {
		return err
	}
	tmp := filepath.Join(os.TempDir(), filepath.Base(path))
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", tmp).Start()
	case "linux":
		return exec.Command("xdg-open", tmp).Start()
	default:
		return exec.Command("open", tmp).Start()
	}
}
