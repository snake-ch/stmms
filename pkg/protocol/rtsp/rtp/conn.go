package rtp

import (
	"gosm/pkg/avformat"
	"gosm/pkg/log"
	"net"
)

const MTU = 1508

// Connection .
type Connection struct {
	goConn     net.PacketConn
	ssrc       uint32
	videoQueue chan *Packet
	audioQueue chan *Packet
	avQueue    chan *avformat.AVPacket
}

// NewConn .
func NewConn(goConn net.PacketConn) *Connection {
	conn := &Connection{
		goConn:     goConn,
		ssrc:       0,
		videoQueue: make(chan *Packet, 512),
		audioQueue: make(chan *Packet, 512),
		avQueue:    make(chan *avformat.AVPacket, 1024),
	}
	return conn
}

// Close .
func (conn *Connection) Close() error {
	return conn.goConn.Close()
}

// return readonly rtp video packet queue
func (conn *Connection) VideoQueue() <-chan *Packet {
	return conn.videoQueue
}

// return readonly rtp video packet queue
func (conn *Connection) AudioQueue() <-chan *Packet {
	return conn.audioQueue
}

// loop to read rtp packet
func (conn *Connection) Serve() {
	defer func() {
		conn.Close()
		log.Debug("RTP: server local: %v, exit", conn.goConn.LocalAddr())
	}()

	for {
		payload := make([]byte, MTU)
		n, _, err := conn.goConn.ReadFrom(payload)
		if err != nil {
			log.Error("RTP: read error, %v", err)
			return
		}
		packet, err := ParsePacket(payload[:n])
		if err != nil {
			log.Error("RTP: parse packet error, %v", err)
			continue
		}

		// one connection supports only one stream type
		if conn.ssrc == 0 {
			conn.ssrc = packet.header.ssrc
		}
		if conn.ssrc != 0 && conn.ssrc != packet.header.ssrc {
			log.Error("RTP: only one stream supported, expected ssrc: %d, but got %d", conn.ssrc, packet.header.ssrc)
			return
		}

		// separate audio/video
		switch packet.header.pt {
		case PacketTypeAVC:
			if len(conn.videoQueue) > cap(conn.videoQueue)-24 {
				log.Debug("RTP: net-stream rtp video packet buffer is nealy full")
			}
			conn.videoQueue <- packet
		case PacketTypeAAC:
			if len(conn.audioQueue) > cap(conn.audioQueue)-24 {
				log.Debug("RTP: net-stream rtp audio packet buffer is nealy full")
			}
			conn.audioQueue <- packet
		}
	}
}
