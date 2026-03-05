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
	"runtime/debug"
	"strings"
	"time"

	"github.com/ndsky1003/crpc/v2"
	"github.com/ndsky1003/crpc/v2/dto"
)

var (
	// Version is set by build flags
	Version = "dev"

	Name   string
	Urls   string
	Root   string
	Secret string
)

func getVersion() string {
	// 1. 如果通过 ldflags 注入了版本，直接使用
	if Version != "dev" {
		return Version
	}

	// 2. 否则尝试从 Go modules 构建信息读取
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	// 3. 如果是通过 go install 安装的，info.Main.Version 会包含版本信息
	//    例如: filemgr-v1.0.0, 或者 devel (如果是本地开发)
	if info.Main.Version != "(devel)" {
		// 从 tool-v1.0.0 格式中提取 v1.0.0
		version := info.Main.Version
		// 检查是否包含 -v（工具名-v版本号）
		if idx := strings.Index(version, "-v"); idx != -1 {
			return version[idx+1:] // 返回 -v 后的部分
		}
		return version
	}

	// 4. 本地开发，返回 dev
	return "dev"
}

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
		fmt.Printf("filemgr version %s\n", getVersion())
		os.Exit(0)
	}
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
