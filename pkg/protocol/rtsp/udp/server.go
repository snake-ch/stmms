package udp

import (
	"context"
	"gosm/pkg/log"
)

// Server .
type Server struct {
	ctx       context.Context
	network   string
	addresses []string
	sessions  []*Session
}

// NewServer .
func NewServer(network string, addresses ...string) (*Server, func(), error) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		ctx:       ctx,
		network:   network,
		addresses: addresses,
		sessions:  make([]*Session, 0),
	}
	closeFunc := func() {
		defer cancel()
		for _, session := range server.sessions {
			if err := session.Close(); err != nil {
				log.Error("%v", err)
			}
		}
	}
	return server, closeFunc, nil
}
