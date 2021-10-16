package rtmp

import (
	"bytes"
	"fmt"

	"gosm/pkg/log"
)

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
		// command response, see:
		// https://help.adobe.com/zh_CN/FlashPlatform/reference/actionscript/3/flash/events/NetStatusEvent.html
		case "_result":
			fallthrough
		case "_error":
			fallthrough
		case "onStatus":
			return nc.onResult(command)
		// commands followed not defined in rtmp-spec-1.0
		case "releaseStream":
		case "FCPublish":
			return nc.onFCPublish(command)
		case "FCUnpublish":
			return nc.onFCUnpublish(command)
		case "deleteStream":
			return nc.onDeleteStream(command)
		case "getStreamLength":
		default:
			return fmt.Errorf("RTMP: unsupport net-connection command type: %s", command.Name)
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
	if obj, ok := command.Objects[0].(map[string]interface{}); ok {
		if app, ok := obj["app"]; ok {
			nc.info.App = app.(string)
		}
		if flashVer, ok := obj["flashVer"]; ok {
			nc.info.FlashVer = flashVer.(string)
		}
		if swfURL, ok := obj["swfUrl"]; ok {
			nc.info.SwfURL = swfURL.(string)
		}
		if tcURL, ok := obj["tcUrl"]; ok {
			nc.info.TcURL = tcURL.(string)
		}
		if fpad, ok := obj["fpad"]; ok {
			nc.info.Fpad = fpad.(bool)
		}
		if capabilities, ok := obj["capabilities"]; ok {
			nc.info.Capabilities = int(capabilities.(float64))
		}
		if audioCodecs, ok := obj["audioCodecs"]; ok {
			nc.info.AudioCodecs = int(audioCodecs.(float64))
		}
		if videoCodecs, ok := obj["videoCodecs"]; ok {
			nc.info.VideoCodecs = int(videoCodecs.(float64))
		}
		if videoFunction, ok := obj["videoFunction"]; ok {
			nc.info.VideoFunction = int(videoFunction.(float64))
		}
		if pageURL, ok := obj["pageUrl"]; ok {
			nc.info.PageURL = pageURL.(string)
		}
		if objectEncoding, ok := obj["objectEncoding"]; ok {
			nc.info.ObjectEncoding = int(objectEncoding.(float64))
		}
	}

	if nc.info.TcURL == "" {
		return nc.WriteCommand(SIDNetConnnection, connectReject(command.TransactionID))
	}

	nc.SetChunkSize(4096)
	nc.SetWindowAckSize()
	nc.SetPeerBandwidth(2500000, BindwidthLimitDynamic)
	return nc.WriteCommand(SIDNetConnnection, connectSuccess(command.TransactionID))
}

// OnCreateStream .
func (nc *NetConnection) onCreateStream(command *Command) error {
	// find unused stream id, got 1 usually
	streamID := uint32(1)
	for {
		if _, ok := nc.streams[streamID]; !ok {
			nc.streams[streamID] = NewNetStream(streamID, nc)
			break
		}
		streamID = streamID + 1
	}
	return nc.WriteCommand(SIDNetConnnection, createStreamSuccess(command.TransactionID, streamID))
}

// FCPublish .
// FME calls FCPublish with the name of the stream whenever a new stream
// is published. This notification can be used by server-side action script
// to maintain list of all streams or also to force FME to stop publishing.
// To stop publishing, call "onFCPublish" with an info object with status
// code set to "NetStream.Publish.BadName".
func (nc *NetConnection) onFCPublish(command *Command) error {
	return nil
}

// fcUnpublish FME notifies server script when a stream is unpublished.
func (nc *NetConnection) onFCUnpublish(command *Command) error {
	return nil
}

// OnDeleteStream .
func (nc *NetConnection) onDeleteStream(command *Command) error {
	// stream id
	if id, ok := command.Objects[1].(float64); ok {
		if stream, ok := nc.streams[uint32(id)]; ok {
			stream.Close()
		}
		delete(nc.streams, uint32(id))
	}
	return nil
}

// OnResult .
func (nc *NetConnection) onResult(command *Command) error {
	return nil
}

// WriteCommand write rtmp command response
// @StreamID: 0 -> netConnection, generate by server -> netStream
func (nc *NetConnection) WriteCommand(streamID uint32, command *Command) error {
	buf := new(bytes.Buffer)
	if _, err := command.WriteTo(buf); err != nil {
		return err
	}
	return nc.WriteMessage(CommandAmf0, streamID, 0, buf.Bytes())
}
