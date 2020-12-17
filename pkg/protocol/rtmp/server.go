package rtmp

import (
	"context"
	"net"
	"time"

	"gosm/pkg/log"
)

// ServerObserver .
type ServerObserver interface {
	OnRTMPPublish(stream *NetStream) error
	OnRTMPUnPublish(stream *NetStream) error
	OnRTMPSubscribe(stream *NetStream) error
	OnRTMPUnSubsribe(stream *NetStream) error
}

// Server rtmp server
type Server struct {
	ctx      context.Context
	network  string
	address  string
	listener net.Listener
	observer ServerObserver
}

// NewServer .
func NewServer(network string, address string, observer ServerObserver) (*Server, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	server := &Server{
		ctx:      ctx,
		network:  network,
		address:  address,
		listener: nil,
		observer: observer,
	}
	closeFunc := func() {
		defer cancel()
		if err := server.listener.Close(); err != nil {
			log.Error("RTMP: server shutdown error, %v", err)
		}
	}
	return server, closeFunc, nil
}

// Start create rtmp connection handle goroutine
func (server *Server) Start() {
	var err error
	server.listener, err = net.Listen(server.network, server.address)
	if err != nil {
		log.Fatal("RTMP: server listen error, %v", err)
	}
	log.Info("RTMP: Server Listen On %s", server.listener.Addr().String())

	go func() {
		for {
			goConn, err := server.listener.Accept()
			if err != nil {
				log.Error("RTMP: server accept error, %v", err)
			} else {
				server.handleConn(goConn)
			}
		}
	}()
}

// handleConn .
func (server *Server) handleConn(goConn net.Conn) {
	log.Debug("RTMP: client remote: %s, server local: %s", goConn.RemoteAddr().String(), goConn.LocalAddr().String())

	// handshake
	rtmpConn := NewNetConn(goConn)
	if err := rtmpConn.ServerHandshake(); err != nil {
		log.Error("RTMP: server handshake error, %v", err)
		rtmpConn.Close()
		return
	}

	// start to serve
	rtmpConn.Start()

	// waiting for publisher or subscriber command
	go func() {
		for {
			select {
			case <-rtmpConn.ctx.Done():
				return
			case stream := <-rtmpConn.streamDone:
				switch stream.status {
				case _publish:
					if err := server.observer.OnRTMPPublish(stream); err != nil {
						log.Error("%v", err)
					}
				case _subscribe:
					if err := server.observer.OnRTMPSubscribe(stream); err != nil {
						log.Error("%v", err)
					}
				case _unpublish:
					if err := server.observer.OnRTMPUnPublish(stream); err != nil {
						log.Error("%v", err)
					}
				case _unsubscribe:
					if err := server.observer.OnRTMPUnSubsribe(stream); err != nil {
						log.Error("%v", err)
					}
				}
			}
		}
	}()
}
