package hls

import (
	"context"
	"gosm/pkg/config"
	"gosm/pkg/log"
	"io/ioutil"
	"net"
	"net/http"
	"path"
	"strings"
)

// Server .
type Server struct {
	ctx      context.Context
	network  string
	address  string
	listener net.Listener
}

// NewServer .
func NewServer(network string, address string) (*Server, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		ctx:      ctx,
		network:  network,
		address:  address,
		listener: nil,
	}

	closeFunc := func() {
		defer cancel()
		if err := server.listener.Close(); err != nil {
			log.Error("%v", err)
		}
	}

	return server, closeFunc, nil
}

// Serve .
func (server *Server) Serve() {
	// listener
	var err error
	server.listener, err = net.Listen(server.network, server.address)
	if err != nil {
		log.Fatal("HLS: server listen error, %v", err)
	}
	log.Info("HLS: server listen on %s", server.listener.Addr().String())

	// muxer
	muxer := http.NewServeMux()
	muxer.HandleFunc("/", server.handleConn)

	// http server
	go func() {
		if err := http.Serve(server.listener, muxer); err != nil {
			log.Error("%v", err)
		}
	}()
}

func (server *Server) handleConn(w http.ResponseWriter, r *http.Request) {
	ext := path.Ext(r.URL.Path)
	if ext != ".m3u8" && ext != ".ts" {
		http.Error(w, "format not support, '.m3u8' or '.ts'", http.StatusBadRequest)
		return
	}

	url := strings.TrimLeft(r.URL.Path, "/")
	urls := strings.Split(url, "/")
	if len(urls) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// stream := urls[len(urls)-2]
	fn := urls[len(urls)-1]
	switch ext {
	case ".m3u8":
		w.Header().Add("Server", config.HLS)
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
		w.Header().Add("Access-Control-Allow-Origin", "*")
	case ".ts":
		w.Header().Add("Server", config.HLS)
		w.Header().Set("Content-Type", "video/MP2T")
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	body, err := ioutil.ReadFile(fn)
	if err != nil {
		http.Error(w, "stream not found", http.StatusNotFound)
		return
	}
	w.Write(body)
}
