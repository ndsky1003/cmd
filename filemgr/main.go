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

	"github.com/ndsky1003/cmd/internal/version"
	"github.com/ndsky1003/crpc/v3"
	"github.com/ndsky1003/crpc/v3/protocol"
)

var (
	// Version is set by build flags
	Version = "dev"
	Name    string
	Urls    string
	Root    string
	CDN     string
)

var (
	NoWeb   bool
	WebAddr string
)

var serverFlag struct {
	IsServer bool
	Secret   string
	Uris     string
}

var clientFlag struct {
	IsClient   bool
	ClientName string
	ClientUris string
}

func init() {
	flag.BoolVar(&serverFlag.IsServer, "server", false, "是否启动server服务器")
	flag.StringVar(&serverFlag.Secret, "serversecret", "", "secret key")
	flag.StringVar(&serverFlag.Uris, "serveruris", ":18083", "server listen uri.eg:127.0.0.1:18083,192.168.2.2:18083")

	flag.StringVar(&Name, "name", "filemgr", "server name, regist in crpc")
	flag.StringVar(&Urls, "urls", "127.0.0.1:18083", "crpc address,eg:127.0.0.1:18083,localhost:18083")
	flag.StringVar(&Root, "root", ".", "root dir")
	flag.StringVar(&CDN, "cdn", "", "cdn base url for file access")
	flag.BoolVar(&NoWeb, "noweb", false, "disable web ui")
	flag.StringVar(&WebAddr, "web", ":18084", "web ui listen address")
	v := flag.Bool("v", false, "print version information and exit")
	flag.BoolVar(v, "version", false, "same as -v")
	flag.Parse()
	if *v {
		fmt.Printf("filemgr version %s\n", version.GetVersion(Version))
		os.Exit(0)
	}
}

func main() {
	if serverFlag.IsServer {
		if serverFlag.Secret == "" {
			randstr := time.Now().UnixNano()
			hash := md5.Sum(fmt.Appendf([]byte{}, "%d", randstr))
			serverFlag.Secret = hex.EncodeToString(hash[:])
		}
		if err := Server(serverFlag.Secret); err != nil {
			panic(fmt.Errorf("crpc server error:%v", err))
		}
	}
	if Urls != "" {
		urls := strings.SplitSeq(Urls, ",")
		for url := range urls {
			fmt.Println("Dial:", url, Name)
			tmpclient, err := crpc.Dial(context.Background(), Name, url, crpc.ClientOptions().SetSecret(Secret))
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
	if !NoWeb {
		if err := startWebServer(WebAddr); err != nil {
			fmt.Println("web ui error:", err)
		}
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
	dir, err := safePath(Root, req.Path)
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
		f := &FileInfo{Name: name, IsDir: file.IsDir(), Cdn: CDN}
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
	dir, err := safePath(Root, req.Path)
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0777)
}

func (*msg) ReadFile(req struct{ Path string }) ([]byte, error) {
	path, err := safePath(Root, req.Path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func (*msg) SaveFile(req *protocol.FileTransfer) (err error) {
	path, err := safePath(Root, req.FileName)
	if err != nil {
		return err
	}
	if req.Offset == 0 {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("file exist:%v", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	flag := os.O_WRONLY | os.O_CREATE
	if req.Offset > 0 {
		flag |= os.O_APPEND
	}
	f, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	_, err = f.Write(req.Data)
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
	for addr := range strings.SplitSeq(Uris, ",") {
		addrs = append(addrs, addr)
	}
	go func() {
		if err := s.Listen(addrs...); err != nil {
			slog.Info("crpc listen error on", "addrs", addrs, "err", err)
		}
	}()
	return nil
}
