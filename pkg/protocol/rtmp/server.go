package rtmp

import (
	"context"
	"errors"
	"fmt"
	"net"

	"gosm/pkg/log"
)

type Observer interface {
	OnRTMPPublish(stream *NetStream) error
	OnRTMPUnPublish(stream *NetStream) error
	OnRTMPSubscribe(stream *NetStream) error
	OnRTMPUnSubsribe(stream *NetStream) error
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

// SetObserver .
func (server *Server) SetObserver(obs Observer) {
	server.obs = obs
}

// Serve .
func (server *Server) Serve() {
	if server.obs == nil {
		log.Fatal("RTMP: observer is empty")
	}

	var err error
	server.listener, err = net.Listen(server.network, server.address)
	if err != nil {
		log.Fatal("RTMP: server listen error, %v", err)
	}
	log.Info("RTMP: server listen on %s", server.listener.Addr())

	go func() {
		for {
			goConn, err := server.listener.Accept()
			if err != nil {
				// close server actived
				if errors.Is(err, net.ErrClosed) {
					return
				}

				log.Error("RTMP: server accept error, %v", err)
				continue
			}
			log.Debug("RTMP: accept remote: %s, local: %s", goConn.RemoteAddr(), goConn.LocalAddr())
			if err := server.handleConn(goConn); err != nil {
				log.Error("%v", err)
			}
		}
	}()
}

// handleConn .
func (server *Server) handleConn(goConn net.Conn) error {
	rtmpConn := NewNetConn(server, goConn)

	if err := rtmpConn.ServerHandshake(); err != nil {
		goConn.Close()
		return fmt.Errorf("RTMP: server handshake error, %w", err)
	}

	return rtmpConn.Serve()
}
