package rtmp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"gosm/pkg/log"
	"gosm/pkg/protocol/amf"
)

func (nc *NetConnection) processMessage(message *Message) error {
	if message == nil {
		return nil
	}
	log.Debug("%0s message: %+v\n", "C -> S", message)

	// check should make acknowledgement
	nc.received += message.Length
	if nc.received >= nc.remoteWindowsSize {
		nc.SetAck(nc.received)
		nc.received = 0
	}

	switch message.TypeID {
	case SetChunkSize:
		return nc.processSetChunkSize(message)
	case AbortMessage:
		return nc.processAbortMessage(message)
	case Acknowledgement:
		return nc.processAcknowledgement(message)
	case UserControlMessages:
		return nc.processUserControlMessages(message)
	case WindowAckSize:
		return nc.processWindowAckSize(message)
	case SetPeerBandwidth:
		return nc.processSetPeerBandwidth(message)
	case AudioType:
		fallthrough
	case VideoType:
		return nc.processMediaMessage(message)
	case AggregateMessageType:
	case SharedObjectAmf0:
	case SharedObjectAmf3:
	case DataAmf0:
		return nc.processMediaMessage(message)
	case DataAmf3:
	case CommandAmf0:
		return nc.processCommandAMF0(message)
	case CommandAmf3:
		return nc.processCommandAMF3(message)
	default:
		log.Debug("unkown message type %d, nothing to do\n", message.TypeID)
	}
	return nil
}

/*******************************************************************
 ************************* Messages Reader *************************
 *******************************************************************/

func (nc *NetConnection) processSetChunkSize(message *Message) error {
	return binary.Read(message.Body, binary.BigEndian, &nc.remoteChunkSize)
}

func (nc *NetConnection) processAbortMessage(message *Message) error {
	return nil
}

func (nc *NetConnection) processAcknowledgement(message *Message) error {
	return nil
}

// processUserControlMessages
//  - EventType (2bytes) | EventData
func (nc *NetConnection) processUserControlMessages(message *Message) error {
	var eventType uint16
	if err := binary.Read(message.Body, binary.BigEndian, &eventType); err != nil {
		return err
	}

	switch eventType {
	case EventStreamBegin:
	case EventStreamEOF:
	case EventStreamDry:
	case EventSetBufferLength:
	case EventStreamIsRecorded:
	case EventPingRequest:
		nc.SetPingResponse(uint32(time.Now().UnixNano() / 1e6))
	case EventPingResponse:
	case EventRequestVerify:
	case EventRespondVerify:
	case EventBufferEmpty:
	case EventBufferReady:
	default:
		return fmt.Errorf("unknown user control message :0x%x", eventType)
	}
	return nil
}

func (nc *NetConnection) processWindowAckSize(message *Message) error {
	if err := binary.Read(message.Body, binary.BigEndian, &nc.remoteWindowsSize); err != nil {
		return err
	}
	return nil
}

func (nc *NetConnection) processSetPeerBandwidth(message *Message) error {
	var err error
	var limit byte
	// bandwidth
	if err = binary.Read(message.Body, binary.BigEndian, &nc.remoteBandwidth); err != nil {
		return err
	}
	// limit
	if limit, err = message.Body.ReadByte(); err != nil {
		return err
	}
	nc.remoteBandwidthLimit = limit

	return nil
}

func (nc *NetConnection) processCommandAMF0(message *Message) error {
	amf := &amf.AMF0{}
	command := &Command{}

	name, err := amf.ReadFrom(message.Body)
	if err != nil {
		return fmt.Errorf("command amf0: error to parse name, %s", err)
	}
	command.Name = name.(string)

	transactionID, err := amf.ReadFrom(message.Body)
	if err != nil {
		return fmt.Errorf("command amf0: error to parse transaction ID, %s", err)
	}
	command.TransactionID = uint32(transactionID.(float64))

	for message.Body.Len() > 0 {
		property, err := amf.ReadFrom(message.Body)
		if err != nil {
			return fmt.Errorf("command amf0: error to parse property, %s", err)
		}
		command.Objects = append(command.Objects, property)
	}

	return nc.onCommand(message.StreamID, command)
}

func (nc *NetConnection) processCommandAMF3(message *Message) error {
	return nil
}

// throw audio/video/metadata message to net-stream
func (nc *NetConnection) processMediaMessage(message *Message) error {
	stream, ok := nc.streams[message.StreamID]
	if !ok {
		return fmt.Errorf("stream id %d not found to transmit audio/video/metadata", message.StreamID)
	}
	stream.mediaQueue <- message
	return nil
}

/*******************************************************************
 ************************** Message Writer *************************
 *******************************************************************/

// WriteMessage .
func (nc *NetConnection) WriteMessage(typeID uint8, streamID uint32, timestamp uint32, body []byte) error {
	message := &Message{
		TypeID:    typeID,
		Length:    0,
		Timestamp: timestamp,
		StreamID:  streamID,
		Body:      nil,
	}

	if body != nil {
		message.Length = uint32(len(body))
		message.Body = bytes.NewBuffer(body)
	} else {
		message.Body = new(bytes.Buffer)
	}

	nc.outMessageStream <- message
	return nil
}

/*******************************************************************
 ********************* Protocol Control Message ********************
 *******************************************************************/

// SetChunkSize .
func (nc *NetConnection) SetChunkSize(size uint32) error {
	message := &Message{
		TypeID:    SetChunkSize,
		Length:    4,
		Timestamp: 0,
		StreamID:  0,
		Body:      new(bytes.Buffer),
	}
	if err := binary.Write(message.Body, binary.BigEndian, size); err != nil {
		return err
	}

	nc.chunkSize = size
	nc.outMessageStream <- message
	return nil
}

// SetAbortMessage .
func (nc *NetConnection) SetAbortMessage(csid uint32) error {
	return nil
}

// SetAck .
func (nc *NetConnection) SetAck(size uint32) error {
	message := &Message{
		TypeID:    Acknowledgement,
		Length:    4,
		Timestamp: 0,
		StreamID:  0,
		Body:      new(bytes.Buffer),
	}
	if err := binary.Write(message.Body, binary.BigEndian, size); err != nil {
		return err
	}

	nc.received = 0
	nc.outMessageStream <- message
	return nil
}

// SetWindowAckSize .
func (nc *NetConnection) SetWindowAckSize() error {
	message := &Message{
		TypeID:    WindowAckSize,
		Length:    4,
		Timestamp: 0,
		StreamID:  0,
		Body:      new(bytes.Buffer),
	}
	if err := binary.Write(message.Body, binary.BigEndian, nc.windowsSize); err != nil {
		return err
	}

	nc.outMessageStream <- message
	return nil
}

// SetPeerBandwidth .
func (nc *NetConnection) SetPeerBandwidth(peerBandwidth uint32, limitType byte) error {
	message := &Message{
		TypeID:    SetPeerBandwidth,
		Length:    5,
		Timestamp: 0,
		StreamID:  0,
		Body:      new(bytes.Buffer),
	}
	if err := binary.Write(message.Body, binary.BigEndian, &peerBandwidth); err != nil {
		return err
	}
	if err := message.Body.WriteByte(limitType); err != nil {
		return err
	}

	nc.bandwidth = peerBandwidth
	nc.bandwithLimit = limitType
	nc.outMessageStream <- message
	return nil
}

/*******************************************************************
 ************************** User Control ***************************
 *******************************************************************/

// UserControlMessage .
func (nc *NetConnection) UserControlMessage(eventType uint16, eventData []byte) error {
	if eventData == nil {
		return fmt.Errorf("error to send user control message, event data is empty")
	}

	message := &Message{
		TypeID:    UserControlMessages,
		Length:    0,
		Timestamp: 0,
		StreamID:  0,
		Body:      new(bytes.Buffer),
	}

	if err := binary.Write(message.Body, binary.BigEndian, eventType); err != nil {
		return err
	}
	message.Length = message.Length + uint32(2)

	if _, err := message.Body.Write(eventData); err != nil {
		return err
	}
	message.Length = message.Length + uint32(len(eventData))

	nc.outMessageStream <- message
	return nil
}

// SetBufferLength user control message set buff length
func (nc *NetConnection) SetBufferLength(streamID uint32, length uint32) error {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[:4], streamID)
	binary.BigEndian.PutUint32(buf[4:], length)
	return nc.UserControlMessage(EventSetBufferLength, buf)
}

// SetStreamBegin user control message stream begin.
func (nc *NetConnection) SetStreamBegin(streamID uint32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, streamID)
	return nc.UserControlMessage(EventStreamBegin, buf)
}

// SetStreamIsRecorded user control message stream is recorded.
func (nc *NetConnection) SetStreamIsRecorded(streamID uint32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, streamID)
	return nc.UserControlMessage(EventStreamIsRecorded, buf)
}

// SetPingResponse user control message to response ping request.
func (nc *NetConnection) SetPingResponse(timestamp uint32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, timestamp)
	return nc.UserControlMessage(EventPingResponse, buf)
}
