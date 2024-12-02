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
	Uris     string
)

func init() {
	flag.BoolVar(&IsServer, "isserver", false, "is server")
	flag.StringVar(&Uris, "uris", ":18083", "server listen uri.eg:127.0.0.1:18083,192.168.2.2:18083")
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
	tmp_path := filepath.Join(Root, req.Filename)
	if req.ChunksIndex == 0 {
		if _, err := os.Stat(tmp_path); err == nil {
			return fmt.Errorf("file exist:%v", tmp_path)
		} else {
			if !os.IsNotExist(err) {
				return err
			}
		}
	}

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
	fmt.Println("server secret:", secret)
	listen_arr := strings.Split(Uris, ",")
	go crpc.NewServer(crpc.OptionServer().SetSecret(secret)).Listens(listen_arr)
}
