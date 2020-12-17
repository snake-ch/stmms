package httpflv

import (
	"context"
	"io"

	"gosm/pkg/avformat"
	"gosm/pkg/avformat/flv"
)

// NetStream implements subscribe interface, play as subscriber
type NetStream struct {
	ctx    context.Context
	cancel context.CancelFunc
	info   *SubscribeInfo
	fw     *flv.Writer
}

// SubscribeInfo .
type SubscribeInfo struct {
	App    string
	Stream string
}

// NewNetStream .
func NewNetStream(w io.Writer, app string, stream string) (*NetStream, error) {
	// flv writer
	fw, err := flv.NewWriter(w, app, stream)
	if err != nil {
		return nil, err
	}
	// http flv media stream
	ctx, cancel := context.WithCancel(context.Background())
	ns := &NetStream{
		ctx:    ctx,
		cancel: cancel,
		info: &SubscribeInfo{
			App:    app,
			Stream: stream,
		},
		fw: fw,
	}
	return ns, nil
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
	flvTag := &flv.Tag{
		TagHeader: &flv.TagHeader{
			TagType:   packet.TypeID,
			DataSize:  packet.Length,
			Timestamp: packet.Timestamp,
			StreamID:  packet.StreamID,
		},
		TagData: packet.Body,
	}
	return ns.fw.WriteTag(flvTag)
}

// Close .
func (ns *NetStream) Close() error {
	ns.cancel()
	return nil
}
