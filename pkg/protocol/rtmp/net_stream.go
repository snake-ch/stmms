package rtmp

import (
	"bytes"
	"fmt"
	"gosm/pkg/avformat"
	"gosm/pkg/log"
)

// stream status
const (
	_StatusUnknown uint8 = iota
	_StatusPublish
	_StatusUnPublish
	_StatusSubscribe
	_StatusUnSubscribe
	_StatusStreamClose
)

// PublishInfo net stream info
type PublishInfo struct {
	Name string
	Type string
}

// SubscribeInfo .
type SubscribeInfo struct {
	StreamName string
	Start      int
	Duration   int
	Reset      bool
}

// NetStream rtmp logical net-stream
type NetStream struct {
	id         uint32
	nc         *NetConnection
	status     uint8
	info       interface{}   // information of publisher or subscriber
	mediaQueue chan *Message // for publishing audio/video/metadata message
}

// NewNetStream return a new rtmp net-stream
func NewNetStream(id uint32, nc *NetConnection) *NetStream {
	return &NetStream{
		id:         id,
		nc:         nc,
		status:     _StatusUnknown,
		info:       nil,
		mediaQueue: make(chan *Message, 1024),
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
		return fmt.Errorf("unsupport rtmp stream command type: %s", command.Name)
	}
	return nil
}

// OnPlay .
func (ns *NetStream) onPlay(command *Command) error {
	if ns.status == _StatusSubscribe || ns.status == _StatusPublish {
		log.Debug("rtmp net-stream id: %d has published or subscribed, ignore Play()", ns.id)
		return nil
	}

	info := &SubscribeInfo{}
	// stream name
	if name, ok := command.Objects[1].(string); ok {
		info.StreamName = name
	}
	// start
	if start, ok := command.Objects[2].(int); ok {
		info.Start = start
	}
	ns.info = info

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
	if err := ns.nc.WriteCommand(ns.id, new(Command).resetStream()); err != nil {
		return err
	}
	// net stream play reset
	if err := ns.nc.WriteCommand(ns.id, new(Command).startStream()); err != nil {
		return err
	}

	// export subscriber
	return ns.streamDone(_StatusSubscribe)
}

// OnPublish .
func (ns *NetStream) onPublish(command *Command) error {
	if ns.status == _StatusSubscribe || ns.status == _StatusPublish {
		log.Debug("rtmp net-stream id: %d has published or subscribed, ignore Publish()", ns.id)
		return nil
	}

	info := &PublishInfo{}
	// stream name
	if name, ok := command.Objects[1].(string); ok {
		info.Name = name
	}
	// stream type
	if t, ok := command.Objects[2].(string); ok {
		info.Type = t
	}
	ns.info = info

	// make response
	if err := ns.nc.WriteCommand(SIDNetStream, new(Command).publishStream()); err != nil {
		return err
	}

	// export publisher
	return ns.streamDone(_StatusPublish)
}

// OnDeleteStream .
func (ns *NetStream) onDeleteStream(command *Command) error {
	return ns.nc.onDeleteStream(command)
}

// make stream done signal: publish, subscribe, unpublish, unsubscribe
func (ns *NetStream) streamDone(status uint8) error {
	if status < _StatusUnknown || status > _StatusStreamClose {
		return fmt.Errorf("net-stream unknown status : %d", status)
	}
	ns.status = status
	ns.nc.streamDone <- ns
	return nil
}

// Info .
func (ns *NetStream) Info() interface{} {
	return ns.info
}

// ConnInfo .
func (ns *NetStream) ConnInfo() *ConnInfo {
	return ns.nc.info
}

/************************************/
/********* Publish Interface ********/
/************************************/

// ReadAVPacket read from publisher, block if no av packet in rtmp net-stream
func (ns *NetStream) ReadAVPacket() (*avformat.AVPacket, error) {
	message := <-ns.mediaQueue
	packet := &avformat.AVPacket{
		TypeID:    message.TypeID,
		Length:    message.Length,
		Timestamp: message.Timestamp,
		StreamID:  message.StreamID,
		Body:      message.Body.Bytes(),
	}
	return packet, nil
}

/************************************/
/******** Subscribe Interface *******/
/************************************/

// WriteAVPacket .
func (ns *NetStream) WriteAVPacket(packet *avformat.AVPacket) error {
	message := &Message{
		TypeID:    packet.TypeID,
		Length:    packet.Length,
		Timestamp: packet.Timestamp,
		StreamID:  ns.id, // uses subscriber's stream id
		Body:      bytes.NewBuffer(packet.Body),
	}

	// check out message buffer is full
	if len(ns.nc.outMessageStream) > cap(ns.nc.outMessageStream)-24 {
		return fmt.Errorf("NetStream: out buffer is full")
	}
	ns.nc.outMessageStream <- message
	return nil
}

// Close .
func (ns *NetStream) Close() error {
	ns.status = _StatusStreamClose
	return nil
}
