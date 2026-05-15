// 文件管理器
// 读
// 写
package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ndsky1003/cmd/common/version"
	"github.com/ndsky1003/crpc/v3"
	"github.com/ndsky1003/crpc/v3/protocol"
)

var (
	// Version is set by build flags
	Version = "dev"
	Secret  string
)

var serverFlag struct {
	IsServer bool
	Secret   string
	Uris     string
}

var clientFlag struct {
	IsClient bool
	Secret   string
	Name     string
	Uris     string
	Root     string
	CDN      string
}

var webFlag struct {
	IsWeb bool
	Uris  string
}

func init() {
	// 1. 创建带 AddSource: true 的选项
	opts := &slog.HandlerOptions{
		AddSource: true, // 启用源文件和行号
		Level:     slog.LevelDebug,
	}

	// 2. 用 TextHandler 输出（key=value 格式）
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	// 或 JSONHandler
	// logger := slog.New(slog.NewJSONHandler(os.Stdout, opts))

	// 3. 设置为全局默认 logger
	slog.SetDefault(logger)

}

func init() {

	flag.BoolVar(&serverFlag.IsServer, "server", false, "是否启动server服务器")
	flag.StringVar(&Secret, "secret", "", "secret key")
	flag.StringVar(&serverFlag.Uris, "suris", ":18083", "server listen uri.eg:127.0.0.1:18083,192.168.2.2:18083")

	flag.BoolVar(&clientFlag.IsClient, "client", false, "是否启动文件服务client")
	flag.StringVar(&clientFlag.Name, "cname", "filemgr", "service name, regist in crpc")
	flag.StringVar(&clientFlag.Uris, "curis", "127.0.0.1:18083", "client dail uri,eg:127.0.0.1:18083,localhost:18083")
	flag.StringVar(&clientFlag.Root, "root", ".", "root dir")
	flag.StringVar(&clientFlag.CDN, "cdn", "", "cdn base url for file access.eg:https://cdn.example.com")

	flag.BoolVar(&webFlag.IsWeb, "web", false, "是否web ui")
	flag.StringVar(&webFlag.Uris, "wuris", ":18084", "web ui listen address,eg:127.0.0.1:18084,localhost:18084")

	v := flag.Bool("v", false, "print version information and exit")
	flag.BoolVar(v, "version", false, "same as -v")
	flag.Parse()

	if Secret != "" {
		serverFlag.Secret = Secret
		clientFlag.Secret = Secret
	}

	if *v {
		fmt.Printf("%s\n", version.GetVersion(Version))
		os.Exit(0)
	}
}

func main() {
	if (serverFlag.IsServer || clientFlag.IsClient) && Secret == "" {
		randstr := time.Now().UnixNano()
		hash := md5.Sum(fmt.Appendf([]byte{}, "%d", randstr))
		Secret = hex.EncodeToString(hash[:])
		serverFlag.Secret = Secret
		clientFlag.Secret = Secret

	}
	if serverFlag.IsServer {
		if err := Server(serverFlag.Secret); err != nil {
			panic(fmt.Errorf("crpc server error:%v", err))
		}
	}
	if clientFlag.IsClient {
		urls := strings.SplitSeq(clientFlag.Uris, ",")
		for url := range urls {
			fmt.Println("Dial:", url, clientFlag.Name)
			tmpclient, err := crpc.Dial(context.Background(), clientFlag.Name, url, crpc.ClientOptions().SetSecret(Secret))
			if err != nil {
				fmt.Println("crpc dial error:", err)
				continue
			}
			if err := tmpclient.RegisterName("crpc", &msg{}); err != nil {
				fmt.Println("crpc register error:", err)
				continue
			}
		}
	}

	if webFlag.IsWeb {
		if err := startWebServer(webFlag.Uris); err != nil {
			fmt.Println("web ui error:", err)
		}
	}
	if !(serverFlag.IsServer || clientFlag.IsClient || webFlag.IsWeb) {
		panic("尚未需要启动的任何服务")
	}
	select {}
}

func safePath(root, reqPath string) (string, error) {
	if reqPath == "" {
		return "", fmt.Errorf("path is empty")
	}
	full := filepath.Join(root, reqPath)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absFull, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absFull, absRoot) {
		return "", fmt.Errorf("access denied: path %q escapes root %q", reqPath, root)
	}
	return absFull, nil
}

type msg struct{}

func (*msg) ListDir(req struct{ Path string }) (res []*FileInfo, err error) {
	slog.Info("list", "path", req.Path)
	defer func() {
		slog.Info("list defer", "path", req.Path)
	}()
	dir, err := safePath(clientFlag.Root, req.Path)
	if err != nil {
		return nil, err
	}
	res = []*FileInfo{}
	fs, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, file := range fs {
		name := file.Name()
		f := &FileInfo{Name: name, IsDir: file.IsDir(), Cdn: clientFlag.CDN}
		if req.Path == "" || req.Path == "/" {
			f.Path = name
		} else {
			f.Path = req.Path + "/" + name
		}
		if !file.IsDir() {
			f.Ext = filepath.Ext(name)
		}
		res = append(res, f)
	}
	return
}

func (*msg) Mkdir(req struct{ Path string }) error {
	dir, err := safePath(clientFlag.Root, req.Path)
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0777)
}

func (*msg) ReadFile(req struct{ Path string }) ([]byte, error) {
	path, err := safePath(clientFlag.Root, req.Path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (*msg) SaveFile(req *protocol.FileTransfer) (err error) {
	slog.Info("SaveFile", "req", req.FileName, "length", len(req.Data))
	defer func() {
		slog.Info("SaveFile defer", "req", req.FileName, "length", len(req.Data))
	}()
	path, err := safePath(clientFlag.Root, req.FileName)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	if len(req.Data) > 0 {
		_, err = f.WriteAt(req.Data, req.Offset)
	}
	return err
}

type FileInfo struct {
	Name  string
	IsDir bool
	Ext   string
	Path  string `json:",omitempty"`
	Cdn   string
}

func (f *FileInfo) String() string {
	return fmt.Sprintf("%+v", *f)
}

func Server(secret string) error {
	fmt.Println("server secret:", secret)
	s, err := crpc.NewServer(context.Background(), crpc.ServerOptions().SetSecret(secret))
	if err != nil {
		return fmt.Errorf("new server: %w", err)
	}

	var addrs []string
	for addr := range strings.SplitSeq(serverFlag.Uris, ",") {
		addrs = append(addrs, addr)
	}
	go func() {
		if err := s.Listen(addrs...); err != nil {
			slog.Info("crpc listen error on", "addrs", addrs, "err", err)
		}
	}()
	return nil
}
