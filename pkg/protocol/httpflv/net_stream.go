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
	app    string
	stream string
	fw     *flv.Writer
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
		app:    app,
		stream: stream,
		fw:     fw,
	}
	return ns, nil
}

/************************************/
/******** Subscribe Interface *******/
/************************************/

// WriteAVPacket .
func (ns *NetStream) WriteAVPacket(packet *avformat.AVPacket) error {
	return ns.fw.WriteRawTag(packet.Body)
}

// Close .
func (ns *NetStream) Close() error {
	ns.cancel()
	return nil
}
