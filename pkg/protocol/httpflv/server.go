package httpflv

import (
	"context"
	"net"
	"net/http"
	"strings"

	"gosm/pkg/log"
)

const (
	_Version = "GOSM/flv_0.0.1"
)

// ServerObserver .
type ServerObserver interface {
	OnHTTPFlvSubscribe(stream *NetStream) error
	OnHTTPFlvUnSubscribe(stream *NetStream) error
}

// Server .
type Server struct {
	ctx      context.Context
	network  string
	address  string
	listener net.Listener
	obs      ServerObserver
}

// NewServer .
func NewServer(network string, address string, obs ServerObserver) (*Server, func(), error) {
	server := &Server{
		network: network,
		address: address,
		obs:     obs,
	}

	var cancel context.CancelFunc
	server.ctx, cancel = context.WithCancel(context.Background())
	closeFunc := func() {
		defer cancel()
		if err := server.listener.Close(); err != nil {
			log.Error("HTTP-Flv: server shutdown error, %v", err)
		}
	}
	return server, closeFunc, nil
}

// Start .
func (server *Server) Start() error {
	// listener
	var err error
	server.listener, err = net.Listen(server.network, server.address)
	if err != nil {
		log.Fatal("HTTP-Flv server listen error, %v", err)
	}
	log.Info("HTTP-Flv Server Listen On %s", server.listener.Addr().String())

	// muxer
	muxer := http.NewServeMux()
	muxer.HandleFunc("/", server.handleConn)

	// http server
	if err := http.Serve(server.listener, muxer); err != nil {
		return err
	}
	return nil
}

// handle stream pulling
func (server *Server) handleConn(w http.ResponseWriter, r *http.Request) {
	// validate url
	if r.Method != "GET" {
		http.Error(w, "method not support", http.StatusBadRequest)
		return
	}

	if !strings.HasSuffix(r.URL.Path, ".flv") {
		http.Error(w, "format not support, only '.flv'", http.StatusBadRequest)
		return
	}

	url := strings.TrimSuffix(strings.TrimLeft(r.URL.Path, "/"), ".flv")
	urls := strings.SplitN(url, "/", 2)
	if len(urls) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	// header response
	w.Header().Add("Server", _Version)
	w.Header().Add("Connection", "Keep-Alive")
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Content-Type", "video/x-flv")
	// w.Header().Add("Content-Type","octet-stream")
	w.Header().Add("Transfer-Encoding", "chunked")
	w.Header().Add("Access-Control-Allow-Origin", "*")
	w.WriteHeader(200)

	// create http-flv media net-stream
	ns, err := NewNetStream(w, urls[0], urls[1])
	if err != nil {
		http.Error(w, "http-flv: create net-stream error", http.StatusInternalServerError)
		return
	}
	if err := server.obs.OnHTTPFlvSubscribe(ns); err != nil {
		http.Error(w, "http-flv: net-stream subscribes error", http.StatusInternalServerError)
		ns.cancel()
	}

	// wait for writing done
	<-ns.ctx.Done()
	if err := server.obs.OnHTTPFlvUnSubscribe(ns); err != nil {
		log.Error("%v", err)
	}
}
