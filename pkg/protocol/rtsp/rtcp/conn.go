package rtcp

import (
	"gosm/pkg/log"
	"net"
)

const MTU = 1508

type Connection struct {
	goConn net.PacketConn
	queue  chan *Packet
}

func NewConn(goConn net.PacketConn) *Connection {
	conn := &Connection{
		goConn: goConn,
		queue:  make(chan *Packet, 32),
	}
	return conn
}

func (conn *Connection) Close() error {
	return conn.goConn.Close()
}

// return readonly rtcp packet queue
func (conn *Connection) Queue() <-chan *Packet {
	return conn.queue
}

// loop to read rtcp packet
func (conn *Connection) Serve() {
	defer func() {
		conn.Close()
		log.Debug("RTCP: connection local: %v, exit", conn.goConn.LocalAddr())
	}()

	for {
		payload := make([]byte, MTU)
		n, _, err := conn.goConn.ReadFrom(payload)
		if err != nil {
			log.Error("RTCP: read error, %v", err)
			continue
		}

		packet, err := ParsePacket(payload[:n])
		if err != nil {
			log.Error("RTCP: parse error, %v", err)
		}
		conn.queue <- packet
	}
}
