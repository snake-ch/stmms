package live

import (
	"time"

	"gosm/pkg/avformat"
)

// Status
const (
	_statusNew = iota
	_statusRunning
	_statusClose
)

// Protocol
const (
	_Rtmp    = "rtmp"
	_HTTPFlv = "http-flv"
	_Hls     = "hls"
	_Dash    = "dash"
)

// Type
const (
	_TypeLive   = "LIVE"
	_TypeRecord = "RECOED"
)

// AVWriteCloser .
type AVWriteCloser interface {
	WriteAVPacket(packet *avformat.AVPacket) error
	Close() error
}

// SubscriberInfo .
type SubscriberInfo struct {
	ID            string
	Protocol      string
	Type          string
	SubscribeTime time.Time
}

// Subscriber .
type Subscriber struct {
	status uint8
	writer AVWriteCloser
	info   *SubscriberInfo
}

func (s *Subscriber) close() error {
	s.status = _statusClose
	return s.writer.Close()
}
