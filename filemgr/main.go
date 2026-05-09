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

	Name   string
	Urls   string
	Root   string
	Secret string
)

var (
	IsServer bool
	Uris     string
)

func init() {
	flag.BoolVar(&IsServer, "isserver", false, "is server")
	flag.StringVar(&Uris, "uris", ":18083", "server listen uri.eg:127.0.0.1:18083,192.168.2.2:18083")
	flag.StringVar(&Name, "name", "filemgr", "server name, regist in crpc")
	flag.StringVar(&Urls, "urls", "127.0.0.1:18083", "crpc address,eg:127.0.0.1:18083,localhost:18083")
	flag.StringVar(&Root, "root", ".", "root dir")
	flag.StringVar(&Secret, "secret", "", "secret key")
	v := flag.Bool("v", false, "print version information and exit")
	flag.BoolVar(v, "version", false, "same as -v")
	flag.Parse()
	if *v {
		fmt.Printf("filemgr version %s\n", version.GetVersion(Version))
		os.Exit(0)
	}
}

func main() {
	if IsServer {
		if Secret == "" {
			randstr := time.Now().UnixNano()
			hash := md5.Sum(fmt.Appendf([]byte{}, "%d", randstr))
			Secret = hex.EncodeToString(hash[:])
		}
		Server(Secret)
	}
	urls := strings.SplitSeq(Urls, ",")
	for url := range urls {
		fmt.Println("Dial:", url, Name)
		tmpclient, err := crpc.Dial(context.Background(), Name, url, crpc.ClientOptions().SetSecret(Secret))
		if err != nil {
			panic(err)
		}
		if err := tmpclient.RegisterName("crpc", &msg{}); err != nil {
			panic(err)
		}
	}
	select {}
}

type msg struct{}

func (*msg) ListDir(req struct{ Path string }) (res []*FileInfo, err error) {
	if strings.HasPrefix(req.Path, ".") {
		err = fmt.Errorf("no access in path {{%v}}", req.Path)
		return
	}
	dir := filepath.Join(Root, req.Path)
	if dir == "" {
		dir = "."
	}
	res = []*FileInfo{}
	fs, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, file := range fs {
		name := file.Name()
		f := &FileInfo{Name: name, IsDir: file.IsDir()}
		if !file.IsDir() {
			f.Ext = filepath.Ext(name)
		}
		res = append(res, f)
	}
	return
}

func (*msg) Mkdir(req struct{ Path string }) error {
	if strings.HasPrefix(req.Path, ".") {
		return fmt.Errorf("no access in path {{%v}}", req.Path)
	}
	if strings.HasPrefix(req.Path, "/") {
		return fmt.Errorf("path:%v not startwith:/", req.Path)
	}
	if req.Path == "" {
		return fmt.Errorf("path is empty")
	}
	dir := filepath.Join(Root, req.Path)
	return os.MkdirAll(dir, 0777)
}

func (*msg) ReadFile(req struct{ Path string }) ([]byte, error) {
	if strings.HasPrefix(req.Path, ".") {
		return nil, fmt.Errorf("no access in path {{%v}}", req.Path)
	}
	return os.ReadFile(filepath.Join(Root, req.Path))
}

func (*msg) SaveFile(req *protocol.FileTransfer) (err error) {
	path := filepath.Join(Root, req.FileName)
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
}

func (f *FileInfo) String() string {
	return fmt.Sprintf("%+v", *f)
}

func Server(secret string) {
	fmt.Println("server secret:", secret)
	s, err := crpc.NewServer(context.Background(), crpc.ServerOptions().SetSecret(secret))
	if err != nil {
		panic(err)
	}
	for addr := range strings.SplitSeq(Uris, ",") {
		addr := addr
		go func() {
			if err := s.Listen(addr); err != nil {
				panic(err)
			}
		}()
	}
}
