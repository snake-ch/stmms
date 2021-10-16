package httpflv

import (
	"context"
	"net"
	"net/http"
	"strings"

	"gosm/pkg/config"
	"gosm/pkg/log"
)

type Observer interface {
	OnHTTPFlvSubscribe(stream *NetStream) error
	OnHTTPFlvUnSubscribe(stream *NetStream) error
}

type Server struct {
	ctx      context.Context
	network  string
	address  string
	listener net.Listener
	obs      Observer
}

// NewServer .
func NewServer(network string, address string) (*Server, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		ctx:      ctx,
		network:  network,
		address:  address,
		listener: nil,
		obs:      nil,
	}

	closeFunc := func() {
		defer cancel()
		if err := server.listener.Close(); err != nil {
			log.Error("%v", err)
		}
	}

	return server, closeFunc, nil
}

// SetObserver
func (server *Server) SetObserver(obs Observer) {
	server.obs = obs
}

// Serve .
func (server *Server) Serve() {
	if server.obs == nil {
		log.Fatal("HTTP-FLV: observer is empty")
	}

	// listener
	var err error
	server.listener, err = net.Listen(server.network, server.address)
	if err != nil {
		log.Fatal("HTTP-FLV: server listen error, %v", err)
	}
	log.Info("HTTP-FLV: server listen on %s", server.listener.Addr().String())

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
	w.Header().Add("Server", config.HTTPFLV)
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

	// waiting for done
	<-ns.ctx.Done()
	if err := server.obs.OnHTTPFlvUnSubscribe(ns); err != nil {
		log.Error("%v", err)
	}
}
