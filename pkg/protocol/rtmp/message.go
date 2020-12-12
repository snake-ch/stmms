package rtmp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
)

// Message RTMP message
type Message struct {
	TypeID    uint8
	Length    uint32
	Timestamp uint32
	StreamID  uint32
	Body      *bytes.Buffer
}

// Remain the length of message remain data
func (message *Message) Remain() uint32 {
	if message.Body == nil {
		return message.Length
	}
	return message.Length - uint32(message.Body.Len())
}

// ReadFrom read chunks combine to message
func (message *Message) ReadFrom(br *bufio.Reader, chunkSize uint32) error {
	remain := message.Remain()
	if remain > chunkSize {
		remain = chunkSize
	}
	for {
		if received, err := io.CopyN(message.Body, br, int64(remain)); err == nil {
			break
		} else if netErr, ok := err.(net.Error); ok {
			return netErr
		} else {
			remain = remain - uint32(received)
		}
	}
	return nil
}

// WriteTo split message to chunks and write to buffer writer
// TODO: support TYPE-1 and TYPE-2 chunks, only support type-0 and type-3 chunk for now
func (message *Message) WriteTo(bw *bufio.Writer, chunkSize uint32) error {
	header := &ChunkHeader{
		timestamp:       message.Timestamp,
		messageLength:   message.Length,
		messageTypeID:   message.TypeID,
		messageStreamID: message.StreamID,
	}

	// chunk stream ID
	switch message.TypeID {
	case SetChunkSize, AbortMessage, Acknowledgement, UserControlMessages, WindowAckSize, SetPeerBandwidth:
		header.chunkStreamID = CSIDProtocolControl
	case CommandAmf0, CommandAmf3:
		header.chunkStreamID = CSIDCommand
	case AudioType:
		header.chunkStreamID = CSIDAudio
	case VideoType:
		header.chunkStreamID = CSIDVideo
	case DataAmf0, DataAmf3:
		header.chunkStreamID = CSIDMetadata
	default:
		return fmt.Errorf("error to write message, no match chunk stream ID")
	}

	// timestamp
	if message.Timestamp >= 0xFFFFFF {
		header.timestamp = 0xFFFFFF
		header.extendedTimestamp = message.Timestamp
	}

	// write first TYPE-0 chunk-header
	header.format = 0
	if err := header.WriteTo(bw); err != nil {
		return err
	}

	if header.messageLength > chunkSize { // multi TYPE-3 chunks
		// first TYPE-0 chunk-data
		_, err := io.CopyN(bw, message.Body, int64(chunkSize))
		if err != nil {
			return err
		}
		// multi TYPE-3 chunks
		remain := header.messageLength - chunkSize

		header.format = 3
		for {
			if err := header.WriteTo(bw); err != nil {
				return err
			}
			if remain > chunkSize {
				_, err := io.CopyN(bw, message.Body, int64(chunkSize))
				if err != nil {
					return err
				}
				remain = remain - chunkSize
			} else {
				_, err := io.CopyN(bw, message.Body, int64(remain))
				if err != nil {
					return err
				}
				break
			}
		}
	} else { // single TYPE-0 chunk
		if _, err := io.CopyN(bw, message.Body, int64(header.messageLength)); err != nil {
			return err
		}
	}
	return nil
}

func (message *Message) String() string {
	return fmt.Sprintf("{TypeID: %d, Length: %d, TimeStramp: %d, StreamID: %d}",
		message.TypeID, message.Length, message.Timestamp, message.StreamID)
}
