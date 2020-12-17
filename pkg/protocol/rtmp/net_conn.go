package rtmp

import (
	"bufio"
	"bytes"
	"context"
	"net"

	"gosm/pkg/log"
)

// NetConnection rtmp logical net connection
type NetConnection struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	goConn               net.Conn
	rw                   *bufio.ReadWriter
	chunkSize            uint32
	remoteChunkSize      uint32
	windowsSize          uint32
	remoteWindowsSize    uint32
	bandwidth            uint32
	remoteBandwidth      uint32
	bandwithLimit        uint8
	remoteBandwidthLimit uint8
	chunkStreams         map[uint32]*ChunkStream
	received             uint32                // received message size
	info                 *ConnInfo             // rtmp connection information
	outMessageStream     chan *Message         // inner writing message channel buffer
	streams              map[uint32]*NetStream // client net-streams
	streamDone           chan *NetStream       // export publisher and subscriber
}

// NewNetConn rtmp logical net connection
func NewNetConn(goConn net.Conn) *NetConnection {
	context, cancelFunc := context.WithCancel(context.Background())
	return &NetConnection{
		ctx:               context,
		cancel:            cancelFunc,
		goConn:            goConn,
		rw:                bufio.NewReadWriter(bufio.NewReader(goConn), bufio.NewWriter(goConn)),
		chunkSize:         128,
		remoteChunkSize:   128,
		windowsSize:       2500000,
		remoteWindowsSize: 2500000,
		chunkStreams:      make(map[uint32]*ChunkStream),
		received:          0,
		info:              nil,
		outMessageStream:  make(chan *Message, 1024),
		streams:           make(map[uint32]*NetStream),
		streamDone:        make(chan *NetStream),
	}
}

// Start start rtmp connection to serve
func (nc *NetConnection) Start() {
	go nc.sending()
	go nc.receiving()
}

// Close close rtmp connection
func (nc *NetConnection) Close() {
	nc.cancel()
	nc.goConn.Close()
}

// Sending loop to send rtmp message
func (nc *NetConnection) sending() {
	defer func() {
		log.Debug("RTMP: client remote: %v, sending exit", nc.goConn.RemoteAddr())
	}()
	for {
		select {
		case <-nc.ctx.Done():
			return
		case message := <-nc.outMessageStream:
			if err := nc.write(message); err != nil {
				log.Error("%v", err)
				return
			}
		}
	}
}

// Receiving loop to receive rtmp message
func (nc *NetConnection) receiving() {
	defer func() {
		nc.Close()
		log.Debug("RTMP: client remote: %v, receiving exit", nc.goConn.RemoteAddr())
	}()

	for {
		message, err := nc.read()
		if err != nil {
			log.Error("RTMP: connection receive error, %v invoke close()", err)
			return
		}
		if err := nc.processMessage(message); err != nil {
			log.Error("RTMP: process message error, %v", err)
			return
		}
	}
}

// read chunk package from client
func (nc *NetConnection) read() (*Message, error) {
	// chunk header
	header := &ChunkHeader{}
	if err := header.ReadFrom(nc.rw.Reader); err != nil {
		return nil, err
	}
	// log.Debug("%0s chunk header: %+v\n", "C -> S", *header)

	// chunk stream
	cs, exist := nc.chunkStreams[header.chunkStreamID]
	if !exist {
		cs = &ChunkStream{
			id:           header.chunkStreamID,
			preHeader:    header,
			cacheMessage: nil,
		}
		nc.chunkStreams[header.chunkStreamID] = cs
	}

	// chunk type
	var message *Message
	switch header.format {
	case 0: // new message
		header.calcTimestamp = header.GetTimestamp()
	case 1: // message with same stream ID
		header.messageStreamID = cs.preHeader.messageStreamID
		header.calcTimestamp = cs.preHeader.calcTimestamp + header.GetTimestamp()
	case 2: // message with same stream ID, message length, message type
		header.messageStreamID = cs.preHeader.messageStreamID
		header.messageLength = cs.preHeader.messageLength
		header.messageTypeID = cs.preHeader.messageTypeID
		header.calcTimestamp = cs.preHeader.calcTimestamp + header.GetTimestamp()
	case 3: // message with same previous header or a continuous chunk
		header.messageStreamID = cs.preHeader.messageStreamID
		header.messageLength = cs.preHeader.messageLength
		header.messageTypeID = cs.preHeader.messageTypeID
		message = cs.cacheMessage
		if message == nil { // new message within chunk size
			switch cs.preHeader.format {
			case 0: // should it happen?
				header.calcTimestamp = cs.preHeader.calcTimestamp
			case 1, 2: // timedelta
				header.calcTimestamp = cs.preHeader.calcTimestamp + cs.preHeader.GetTimestamp()
			}
		} else { // continuous message
			header.calcTimestamp = cs.preHeader.calcTimestamp
		}
	}
	cs.preHeader = header

	// chunk data
	if message == nil {
		message = &Message{
			TypeID:    header.messageTypeID,
			Length:    header.messageLength,
			Timestamp: header.calcTimestamp,
			StreamID:  header.messageStreamID,
			Body:      new(bytes.Buffer),
		}
	}
	if err := message.ReadFrom(nc.rw.Reader, nc.remoteChunkSize); err != nil {
		return nil, err
	}
	if message.Remain() == 0 {
		cs.cacheMessage = nil
		return message, nil
	}
	cs.cacheMessage = message
	return nil, nil
}

// write message to client
func (nc *NetConnection) write(message *Message) error {
	// log.Debug("%0s message: %+v\n", "S -> C", message)

	if err := message.WriteTo(nc.rw.Writer, nc.chunkSize); err != nil {
		return err
	}
	if err := nc.rw.Flush(); err != nil {
		return err
	}
	return nil
}
