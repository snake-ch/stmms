package hls

import (
	"context"
	"errors"
	"gosm/pkg/avformat"
	"gosm/pkg/log"
)

// NetStream implements subscribe interface, play as subscriber
type NetStream struct {
	ctx       context.Context
	cancel    context.CancelFunc
	info      *SubscribeInfo
	w         *Writer
	outBuffer chan *avformat.AVPacket
}

// SubscribeInfo .
type SubscribeInfo struct {
	App    string
	Stream string
}

func NewNetStream(app string, stream string) (*NetStream, error) {
	// ts writer
	w, err := NewWriter(stream)
	if err != nil {
		return nil, err
	}
	// hls media stream
	ctx, cancel := context.WithCancel(context.Background())
	ns := &NetStream{
		ctx:    ctx,
		cancel: cancel,
		info: &SubscribeInfo{
			App:    app,
			Stream: stream,
		},
		w:         w,
		outBuffer: make(chan *avformat.AVPacket, 1024),
	}
	go ns.writting()

	return ns, nil
}

func (ns *NetStream) writting() {
	for {
		select {
		case <-ns.ctx.Done():
			return
		case packet := <-ns.outBuffer:
			if err := ns.w.Write(packet); err != nil {
				log.Error("HLS: write ts file error, %v", err)
				return
			}
		}
	}
}

/************************************/
/******** Subscribe Interface *******/
/************************************/

// Info .
func (ns *NetStream) Info() *SubscribeInfo {
	return ns.info
}

// WriteAVPacket .
func (ns *NetStream) WriteAVPacket(packet *avformat.AVPacket) error {
	if len(ns.outBuffer) > cap(ns.outBuffer)-24 {
		return errors.New("RTMP: net-stream out buffer is full")
	}
	ns.outBuffer <- packet
	return nil
}

// Close .
func (ns *NetStream) Close() error {
	return ns.w.tsMuxer.Close()
}
