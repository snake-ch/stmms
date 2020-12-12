package rtmp

import (
	"bytes"
	"fmt"

	"gosm/pkg/log"
)

// ConnInfo net connection information
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

// OnCommand rtmp command handler
func (nc *NetConnection) onCommand(streamID uint32, command *Command) error {
	log.Debug("%0s command: %+v\n", "C -> S", *command)

	// by convention, stream id for net-connection command equals 0
	if streamID == 0 {
		switch command.Name {
		// commands followed defined in rtmp-spec-1.0, see:
		// http://wwwimages.adobe.com/content/dam/Adobe/en/devnet/rtmp/pdf/rtmp_specification_1.0.pdf
		case "connect":
			return nc.onConnect(command)
		case "call":
		case "close":
		case "createStream":
			return nc.onCreateStream(command)
		// commands followed not defined in rtmp_spec, but using stream which id equals 0
		case "releaseStream":
		case "FCPublish":
		case "FCUnpublish":
		case "deleteStream":
			return nc.onDeleteStream(command)
		case "getStreamLength":
		default:
			return fmt.Errorf("unsupport rtmp net-connection command type: %s", command.Name)
		}
	}

	// by convention, stream id for net-stream command equals 1
	if stream, ok := nc.streams[streamID]; ok {
		return stream.onCommand(command)
	}
	return nil
}

// OnConnect .
func (nc *NetConnection) onConnect(command *Command) error {
	// parse client information
	nc.info = &ConnInfo{}
	if property, ok := command.Objects[0].(map[string]interface{}); ok {
		if app, ok := property["app"]; ok {
			nc.info.App = app.(string)
		}
		if flashVer, ok := property["flashVer"]; ok {
			nc.info.FlashVer = flashVer.(string)
		}
		if swfURL, ok := property["swfUrl"]; ok {
			nc.info.SwfURL = swfURL.(string)
		}
		if tcURL, ok := property["tcUrl"]; ok {
			nc.info.SwfURL = tcURL.(string)
		}
		if fpad, ok := property["fpad"]; ok {
			nc.info.Fpad = fpad.(bool)
		}
		if capabilities, ok := property["capabilities"]; ok {
			nc.info.Capabilities = int(capabilities.(float64))
		}
		if audioCodecs, ok := property["audioCodecs"]; ok {
			nc.info.AudioCodecs = int(audioCodecs.(float64))
		}
		if videoCodecs, ok := property["videoCodecs"]; ok {
			nc.info.VideoCodecs = int(videoCodecs.(float64))
		}
		if videoFunction, ok := property["videoFunction"]; ok {
			nc.info.VideoFunction = int(videoFunction.(float64))
		}
		if objectEncoding, ok := property["objectEncoding"]; ok {
			nc.info.ObjectEncoding = int(objectEncoding.(float64))
		}
	}
	nc.SetChunkSize(4096)
	nc.SetWindowAckSize()
	nc.SetPeerBandwidth(2500000, BindwidthLimitDynamic)
	return nc.WriteCommand(SIDNetConnnection, new(Command).connectSuccess(command.TransactionID))
}

// OnCreateStream .
func (nc *NetConnection) onCreateStream(command *Command) error {
	if nc.info == nil {
		return fmt.Errorf("rtmp net-connection command: %s error, connect server before", command.Name)
	}
	// find unused stream id, got 1 usually
	streamID := uint32(1)
	for {
		if _, ok := nc.streams[streamID]; !ok {
			nc.streams[streamID] = NewNetStream(streamID, nc)
			break
		}
		streamID = streamID + 1
	}
	return nc.WriteCommand(SIDNetConnnection, new(Command).createStreamSuccess(command.TransactionID, streamID))
}

// OnDeleteStream .
func (nc *NetConnection) onDeleteStream(command *Command) error {
	// stream id
	if id, ok := command.Objects[1].(float64); !ok {
		log.Error("net-connection command: %s error, stream %d not found", command.Name, uint32(id))
	} else {
		delete(nc.streams, uint32(id))
	}
	return nil
}

// WriteCommand write rtmp command response
// @StreamID: 0 -> netConnection, generate by server -> netStream
func (nc *NetConnection) WriteCommand(streamID uint32, command *Command) error {
	buf := new(bytes.Buffer)
	command.WriteTo(buf)
	return nc.WriteMessage(CommandAmf0, streamID, 0, buf.Bytes())
}
