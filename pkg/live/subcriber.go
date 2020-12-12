package live

import (
	"time"

	"gosm/pkg/avformat"
)

// Protocol
const (
	_Rtmp    = "rtmp"
	_HTTPFlv = "http-flv"
	_Hls     = "hls"
	_Dash    = "dash"
)

const (
	_TypeLive   = "LIVE"
	_TypeRecord = "RECOED"
)

// AVWriteCloser .
type AVWriteCloser interface {
	WriteAVPacket(packet *avformat.AVPacket) error
	Close() error
}

// Subscriber .
type Subscriber struct {
	stream AVWriteCloser
	info   *SubscriberInfo
}

// SubscriberInfo .
type SubscriberInfo struct {
	ID            string
	Protocol      string
	Type          string
	SubscribeTime time.Time
}
