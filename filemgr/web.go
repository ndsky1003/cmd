package main

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ndsky1003/crpc/v3"
	"github.com/ndsky1003/crpc/v3/coder"
	"github.com/ndsky1003/crpc/v3/protocol"
	netclient "github.com/ndsky1003/net/v2/client"
	netconn "github.com/ndsky1003/net/v2/conn"
)

//go:embed web_index.html
var webFS embed.FS

type serverConn struct {
	Name    string
	Addr    string
	Secret  string
	Service string
	client  *crpc.Client
}

var (
	mu             sync.RWMutex
	sessionServers = map[string]map[string]*serverConn{} // token -> name -> conn
)

func sessionMap(token string) map[string]*serverConn {
	mu.RLock()
	m, ok := sessionServers[token]
	mu.RUnlock()
	if !ok {
		m = map[string]*serverConn{}
		mu.Lock()
		sessionServers[token] = m
		mu.Unlock()
	}
	return m
}

func tokenFrom(r *http.Request) string {
	if t := r.URL.Query().Get("_token"); t != "" {
		return t
	}
	return ""
}

func startWebServer(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/token", handleToken)
	mux.HandleFunc("/api/servers", handleServers)
	mux.HandleFunc("/api/connect", handleConnect)
	mux.HandleFunc("/api/disconnect", handleDisconnect)
	mux.HandleFunc("/api/list", handleList)
	mux.HandleFunc("/api/mkdir", handleMkdir)
	mux.HandleFunc("/api/upload", handleUpload)
	mux.HandleFunc("/api/read", handleRead)
	mux.HandleFunc("/", handleIndex)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("web listen %s: %w", addr, err)
	}
	fmt.Println("web ui  : http://" + addr)
	go http.Serve(ln, mux)
	return nil
}

func handleIndex(rw http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.Error(rw, "not found", 404)
		return
	}
	data, err := webFS.ReadFile("web_index.html")
	if err != nil {
		http.Error(rw, err.Error(), 500)
		return
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	rw.Write(data)
}

func handleToken(rw http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)
	sessionMap(token) // ensure map exists
	writeJSON(rw, map[string]string{"token": token})
}

func getServer(token, name string) (*serverConn, error) {
	m := sessionMap(token)
	mu.RLock()
	s, ok := m[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("server %q not found", name)
	}
	return s, nil
}

func call(s *serverConn, method string, args, reply any) error {
	return s.client.Call(context.Background(), s.Service, "crpc."+method, args, reply)
}

func handleServers(rw http.ResponseWriter, r *http.Request) {
	token := tokenFrom(r)
	m := sessionMap(token)
	mu.RLock()
	list := make([]map[string]string, 0)
	for _, s := range m {
		list = append(list, map[string]string{
			"name": s.Name, "addr": s.Addr, "service": s.Service,
		})
	}
	mu.RUnlock()
	writeJSON(rw, list)
}

func handleConnect(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(rw, "method not allowed", 405)
		return
	}
	var req struct {
		Token   string `json:"token"`
		Name    string `json:"name"`
		Addr    string `json:"addr"`
		Secret  string `json:"secret"`
		Service string `json:"service"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	if len(req.Token) < 8 {
		http.Error(rw, "invalid token", 400)
		return
	}
	if req.Name == "" {
		req.Name = "filemgr"
	}
	if req.Service == "" {
		req.Service = "filemgr"
	}
	if req.Addr == "" {
		http.Error(rw, "addr is required", 400)
		return
	}
	if h, p, err := net.SplitHostPort(req.Addr); err != nil || p == "" {
		http.Error(rw, "invalid address, need host:port", 400)
		return
	} else if h == "" {
		req.Addr = "127.0.0.1:" + p
	}

	// fast tcp probe
	dialer := net.Dialer{Timeout: 3 * time.Second}
	if probe, err := dialer.DialContext(context.Background(), "tcp", req.Addr); err != nil {
		http.Error(rw, "connect failed: "+err.Error(), 400)
		return
	} else {
		probe.Close()
	}

	peerName := "web_" + req.Token[:8]
	client, err := crpc.Dial(context.Background(), peerName, req.Addr,
		crpc.ClientOptions().SetSecret(req.Secret).
			WithConn(func(o *netclient.Option) {
				o.WithConn(func(oo *netconn.Option) {
					oo.SetReadBufferLimitSize(500 * 1024 * 1024)
					oo.SetWriteTimeout(60 * time.Second)
					oo.SetSendChanTimeout(60 * time.Second)
					oo.SetHeartInterval(15 * time.Second)
				})
			}),
	)
	if err != nil {
		http.Error(rw, "connect failed: "+err.Error(), 400)
		return
	}

	// wait for crpc ready (up to 5s)
	ready := make(chan error, 1)
	go func() {
		for range 10 {
			var res []any
			if err := client.Call(context.Background(), req.Service, "crpc.ListDir", struct{ Path string }{Path: "/"}, &res); err == nil {
				ready <- nil
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		ready <- fmt.Errorf("connection timeout")
	}()
	select {
	case err := <-ready:
		if err != nil {
			client.Close()
			http.Error(rw, "connect failed: "+err.Error(), 400)
			return
		}
	case <-time.After(5 * time.Second):
		client.Close()
		http.Error(rw, "connect timeout", 400)
		return
	}

	s := &serverConn{
		Name: req.Name, Addr: req.Addr, Secret: req.Secret, Service: req.Service, client: client,
	}
	m := sessionMap(req.Token)
	mu.Lock()
	if s, ok := m[req.Name]; ok {
		s.client.Close()
		delete(m, req.Name)
	}
	m[req.Name] = s
	mu.Unlock()
	writeJSON(rw, map[string]string{"ok": "connected", "name": req.Name})
}

func handleDisconnect(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(rw, "method not allowed", 405)
		return
	}
	var req struct {
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	m := sessionMap(req.Token)
	mu.Lock()
	if s, ok := m[req.Name]; ok {
		s.client.Close()
		delete(m, req.Name)
	}
	mu.Unlock()
	writeJSON(rw, map[string]string{"ok": "disconnected"})
}

type fileInfo struct {
	Name  string `json:"Name"`
	IsDir bool   `json:"IsDir"`
	Ext   string `json:"Ext"`
	Path  string `json:"Path"`
	Cdn   string `json:"Cdn"`
}

func handleList(rw http.ResponseWriter, r *http.Request) {
	token := tokenFrom(r)
	srv := r.URL.Query().Get("s")
	path := r.URL.Query().Get("path")
	s, err := getServer(token, srv)
	if err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	var res []*fileInfo
	if err := call(s, "ListDir", struct{ Path string }{Path: path}, &res); err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	writeJSON(rw, res)
}

func handleMkdir(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(rw, "method not allowed", 405)
		return
	}
	token := tokenFrom(r)
	var req struct {
		S    string `json:"s"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	s, err := getServer(token, req.S)
	if err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	if err := call(s, "Mkdir", struct{ Path string }{Path: req.Path}, nil); err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	writeJSON(rw, map[string]string{"ok": "created"})
}

func handleUpload(rw http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(rw, "method not allowed", 405)
		return
	}
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	token := tokenFrom(r)
	srv := r.FormValue("s")
	dir := r.FormValue("dir")
	s, err := getServer(token, srv)
	if err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	defer file.Close()

	filename := header.Filename
	if dir != "" {
		filename = dir + "/" + filename
	}

	const chunkSize = 1024 * 1024
	buf := make([]byte, chunkSize)
	offset := int64(0)
	bytesSincePause := int64(0)
	for {
		n, _ := io.ReadFull(file, buf)
		if n == 0 {
			break
		}
		data := buf[:n]
		offset += int64(n)
		ft := &protocol.FileTransfer{FileName: filename, Data: data, Offset: offset - int64(n), IsFinish: false}
		slog.Info("call", "filename", ft.FileName, "offset", ft.Offset)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.client.Call(ctx, s.Service, "crpc.SaveFile", ft,
			crpc.ClientOptions().SetReqCoderT(coder.Msgp).
				SetDebug(true)); err != nil {
			http.Error(rw, err.Error(), 400)
			return
		}
		slog.Info("call after", "filename", ft.FileName, "offset", ft.Offset)
		bytesSincePause += int64(n)
		if bytesSincePause >= 16*1024*1024 {
			time.Sleep(200 * time.Millisecond)
			bytesSincePause = 0
		}
	}

	slog.Info("1")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ft := &protocol.FileTransfer{FileName: filename, Offset: offset, IsFinish: true}
	if err := s.client.Call(ctx, s.Service, "crpc.SaveFile", ft, nil); err != nil {
		slog.Info("2", "err", err)
		http.Error(rw, err.Error(), 400)
		return
	}
	slog.Info("3")

	writeJSON(rw, map[string]string{"ok": "saved"})
}

var mimeTypes = map[string]string{
	".txt": "text/plain; charset=utf-8", ".md": "text/markdown; charset=utf-8",
	".json": "application/json; charset=utf-8", ".xml": "application/xml; charset=utf-8",
	".yaml": "text/plain; charset=utf-8", ".yml": "text/plain; charset=utf-8",
	".toml": "text/plain; charset=utf-8", ".csv": "text/plain; charset=utf-8",
	".go": "text/plain; charset=utf-8", ".py": "text/plain; charset=utf-8",
	".js": "text/plain; charset=utf-8", ".ts": "text/plain; charset=utf-8",
	".css": "text/plain; charset=utf-8", ".html": "text/plain; charset=utf-8",
	".sh": "text/plain; charset=utf-8", ".log": "text/plain; charset=utf-8",
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png",
	".gif": "image/gif", ".svg": "image/svg+xml", ".bmp": "image/bmp",
	".webp": "image/webp", ".ico": "image/x-icon",
	".pdf": "application/pdf",
	".mp4": "video/mp4", ".webm": "video/webm", ".ogg": "video/ogg",
	".mov": "video/quicktime", ".avi": "video/x-msvideo", ".mkv": "video/x-matroska",
}

func handleRead(rw http.ResponseWriter, r *http.Request) {
	token := tokenFrom(r)
	srv := r.URL.Query().Get("s")
	path := r.URL.Query().Get("path")
	s, err := getServer(token, srv)
	if err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	inline := r.URL.Query().Get("inline") == "1"
	if ct, ok := mimeTypes[ext]; ok && inline {
		rw.Header().Set("Content-Type", ct)
	} else {
		name := filepath.Base(path)
		rw.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		rw.Header().Set("Content-Type", "application/octet-stream")
	}

	readCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var data []byte
	if err := s.client.Call(readCtx, s.Service, "crpc.ReadFile", struct{ Path string }{Path: path}, &data); err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	rw.Write(data)
}

func writeJSON(rw http.ResponseWriter, v any) {
	data, _ := json.Marshal(v)
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.Write(data)
}
