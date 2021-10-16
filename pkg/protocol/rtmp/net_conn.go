package rtmp

import (
	"bufio"
	"bytes"
	"errors"
	"net"

	"gosm/pkg/log"
)

// ConnInfo see rtmp-sepc-1.0 section 7.2.1.1 connect
type ConnInfo struct {
	App            string
	FlashVer       string
	SwfURL         string
	TcURL          string
	Fpad           bool
	Capabilities   int
	AudioCodecs    int
	VideoCodecs    int
	VideoFunction  int
	PageURL        string
	ObjectEncoding int
}

// NetConnection rtmp logical net connection
type NetConnection struct {
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
	outBuffer            chan *Message         // inner writing message buffer
	streams              map[uint32]*NetStream // client net-streams
	server               *Server               // rtmp server, for publishing callback
}

// NewNetConn rtmp logical net connection
func NewNetConn(server *Server, goConn net.Conn) *NetConnection {
	return &NetConnection{
		goConn:            goConn,
		rw:                bufio.NewReadWriter(bufio.NewReader(goConn), bufio.NewWriter(goConn)),
		chunkSize:         128,
		remoteChunkSize:   128,
		windowsSize:       2500000,
		remoteWindowsSize: 2500000,
		chunkStreams:      make(map[uint32]*ChunkStream),
		received:          0,
		info:              &ConnInfo{},
		outBuffer:         make(chan *Message, 1024),
		streams:           make(map[uint32]*NetStream),
		server:            server,
	}
}

// Close close rtmp connection
func (nc *NetConnection) Close() error {
	return nc.goConn.Close()
}

// Serve .
func (nc *NetConnection) Serve() error {
	// do loop to read rtmp message
	go func() {
		defer func() {
			nc.Close()
			log.Debug("RTMP: client remote: %v, reading exit", nc.goConn.RemoteAddr())
		}()

		for {
			message, err := nc.Read()
			if err != nil {
				log.Error("RTMP: net connection read error, %v", err)
				return
			}

			if message == nil {
				continue
			}

			if err := nc.process(message); err != nil {
				log.Error("RTMP: process message error, %v", err)
				return
			}
		}
	}()

	// do loop to write rtmp message
	go func() {
		defer func() {
			log.Debug("RTMP: client remote: %v, writing exit", nc.goConn.RemoteAddr())
		}()

		for message := range nc.outBuffer {
			if err := nc.Write(message); err != nil {
				log.Error("%v", err)
				return
			}
		}
	}()

	return nil
}

// read full message from client, return nil if got a chunk
func (nc *NetConnection) Read() (*Message, error) {
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
func (nc *NetConnection) Write(message *Message) error {
	// log.Debug("%0s message: %+v\n", "S -> C", message)
	if err := message.WriteTo(nc.rw.Writer, nc.chunkSize); err != nil {
		return err
	}
	if err := nc.rw.Flush(); err != nil {
		return err
	}
	return nil
}

// write message to net-connection inner buffer
func (nc *NetConnection) AsyncWrite(message *Message) error {
	if len(nc.outBuffer) > cap(nc.outBuffer)-24 {
		return errors.New("RTMP: net-stream inner out buffer is full")
	}
	nc.outBuffer <- message
	return nil
}
