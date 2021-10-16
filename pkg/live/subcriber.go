package live

import (
	"time"

	"gosm/pkg/avformat"
)

// Status
const (
	New = iota + 1
	Running
	Closed
)

// Protocol
const (
	RTMP    = "rtmp"
	HTTPFLV = "http-flv"
	HLS     = "hls"
	DASH    = "dash"
)

// Type
const (
	TypeLive   = "LIVE"
	TypeRecord = "RECOED"
)

// AVWriteCloser .
type AVWriteCloser interface {
	WriteAVPacket(packet *avformat.AVPacket) error
	Close() error
}

// SubscriberInfo .
type SubscriberInfo struct {
	UID           string
	Protocol      string
	Type          string
	SubscribeTime time.Time
}

// Subscriber .
type Subscriber struct {
	status uint8
	wc     AVWriteCloser
	info   *SubscriberInfo
}

func (s *Subscriber) Close() error {
	s.status = Closed
	return s.wc.Close()
}
