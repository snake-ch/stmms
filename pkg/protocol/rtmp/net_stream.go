package rtmp

import (
	"bytes"
	"fmt"
	"time"

	"gosm/pkg/avformat"
	"gosm/pkg/config"
)

var AVReadTimeout = time.Duration(config.Global.RTMP.AVReadTimeout) * time.Second

// StreamInfo
type StreamInfo struct {
	// see rtmp-spec-1.0 netstream command Publish()
	Name string
	Type string

	// see rtmp-spec-1.0 netstream command Play()
	StreamName string
	Start      int
	Duration   int
	Reset      bool
}

// NetStream rtmp logical net-stream
type NetStream struct {
	id      uint32
	nc      *NetConnection
	info    *StreamInfo   // information of publisher or subscriber
	avQueue chan *Message // for publishing audio/video/metadata message
	timer   *time.Timer   // for read timeout
	closed  bool
}

// NewNetStream return a new rtmp net-stream
func NewNetStream(id uint32, nc *NetConnection) *NetStream {
	return &NetStream{
		id:      id,
		nc:      nc,
		info:    &StreamInfo{},
		closed:  false,
		timer:   nil,
		avQueue: make(chan *Message, 1024),
	}
}

// OnCommand rtmp stream command handler
func (ns *NetStream) onCommand(command *Command) error {
	switch command.Name {
	case "play":
		return ns.onPlay(command)
	case "play2":
	case "deleteStream":
		return ns.onDeleteStream(command)
	case "closeStream":
	case "recevieAudio":
	case "recevieVideo":
	case "publish":
		return ns.onPublish(command)
	case "seek":
	case "pause":
	default:
		return fmt.Errorf("RTMP: unsupport net-stream command type: %s", command.Name)
	}
	return nil
}

// OnPlay .
func (ns *NetStream) onPlay(command *Command) error {
	// stream name
	if name, ok := command.Objects[1].(string); ok {
		ns.info.StreamName = name
	}
	// start
	if start, ok := command.Objects[2].(int); ok {
		ns.info.Start = start
	}

	// set chunksize
	if err := ns.nc.SetChunkSize(ns.nc.chunkSize); err != nil {
		return err
	}
	// stream is recorded
	if err := ns.nc.SetStreamIsRecorded(ns.id); err != nil {
		return err
	}
	// stream begin
	if err := ns.nc.SetStreamBegin(ns.id); err != nil {
		return err
	}
	// net stream play start
	if err := ns.nc.WriteCommand(ns.id, resetStream()); err != nil {
		return err
	}
	// net stream play reset
	if err := ns.nc.WriteCommand(ns.id, startStream()); err != nil {
		return err
	}

	// export subscriber
	if err := ns.nc.server.obs.OnRTMPSubscribe(ns); err != nil {
		return err
	}

	return nil
}

// OnPublish .
func (ns *NetStream) onPublish(command *Command) error {
	// stream name
	if name, ok := command.Objects[1].(string); ok {
		ns.info.Name = name
	}
	// stream type
	if t, ok := command.Objects[2].(string); ok {
		ns.info.Type = t
	}

	// make response
	if err := ns.nc.WriteCommand(SIDNetStream, publishStream()); err != nil {
		return err
	}

	// export publisher
	ns.timer = time.NewTimer(AVReadTimeout)
	if err := ns.nc.server.obs.OnRTMPPublish(ns); err != nil {
		return err
	}

	return nil
}

// OnDeleteStream .
func (ns *NetStream) onDeleteStream(command *Command) error {
	return ns.nc.onDeleteStream(command)
}

// Info .
func (ns *NetStream) Info() *StreamInfo {
	return ns.info
}

// ConnInfo .
func (ns *NetStream) ConnInfo() *ConnInfo {
	return ns.nc.info
}

/************************************/
/********* Publish Interface ********/
/************************************/

// ReadAVPacket read with timeout
func (ns *NetStream) ReadAVPacket() (*avformat.AVPacket, error) {
	if ns.closed {
		return nil, fmt.Errorf("RTMP: stream '%s' is closed", ns.info.Name)
	}

	ns.timer.Reset(AVReadTimeout)
	select {
	case <-ns.timer.C:
		return nil, fmt.Errorf("RTMP: stream '%s' read timeout", ns.info.Name)
	case message, ok := <-ns.avQueue:
		if !ok {
			return nil, fmt.Errorf("RTMP: stream '%s' media buffer closed", ns.info.Name)
		}
		packet := &avformat.AVPacket{
			TypeID:    message.TypeID,
			Length:    message.Length,
			Timestamp: message.Timestamp,
			StreamID:  message.StreamID,
			Body:      message.Body.Bytes(),
		}
		return packet, nil
	}
}

/************************************/
/******** Subscribe Interface *******/
/************************************/

// WriteAVPacket non-block write
func (ns *NetStream) WriteAVPacket(packet *avformat.AVPacket) error {
	if ns.closed {
		return fmt.Errorf("RTMP: stream id '%d' is closed", ns.id)
	}

	message := &Message{
		TypeID:    packet.TypeID,
		Length:    packet.Length,
		Timestamp: packet.Timestamp,
		StreamID:  ns.id, // uses subscriber's stream id
		Body:      bytes.NewBuffer(packet.Body),
	}

	return ns.nc.AsyncWrite(message)
}

// Close .
func (ns *NetStream) Close() error {
	ns.closed = true
	ns.nc.Close()
	return nil
}
