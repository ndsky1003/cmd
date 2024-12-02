// 文件管理器
// 读
// 写
package main

import (
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ndsky1003/crpc/v2"
	"github.com/ndsky1003/crpc/v2/dto"
)

var (
	Name   string
	Urls   string
	Root   string
	Secret string
)

var (
	IsServer bool
	Port     int
)

func init() {
	flag.BoolVar(&IsServer, "isserver", false, "is server")
	flag.IntVar(&Port, "port", 18083, "server port")
	flag.StringVar(&Name, "name", "filemgr", "server name, regist in crpc")
	flag.StringVar(&Urls, "urls", "127.0.0.1:18083", "crpc address,eg:127.0.0.1:18083,localhost:18083")
	flag.StringVar(&Root, "root", ".", "root dir")
	flag.StringVar(&Secret, "secret", "", "secret key")
	flag.Parse()
}

func main() {
	if IsServer {
		if Secret == "" {
			randstr := time.Now().UnixNano()
			hash := md5.Sum([]byte(fmt.Sprintf("%d", randstr)))
			Secret = hex.EncodeToString(hash[:])
		}
		Server(Secret)
	}
	urls := strings.Split(Urls, ",")
	for _, url := range urls {
		fmt.Println("Dial:", url, Name)
		tmpclient := crpc.Dial(Name, url, crpc.Options().SetSecret(Secret))
		tmpclient.RegisterName("crpc", &msg{})
	}
	select {}
}

type msg struct {
}

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

func (*msg) SaveFile(req *dto.FileBody) error {
	return crpc.WriteFile(req)
}

type FileInfo struct {
	Name  string
	IsDir bool
	Ext   string
}

func (this *FileInfo) String() string {
	return fmt.Sprintf("%+v", *this)
}

func Server(secret string) {
	go crpc.NewServer(crpc.OptionServer().SetSecret(secret)).Listen(fmt.Sprintf(":%v", Port))
}
