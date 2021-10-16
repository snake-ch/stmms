package live

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"gosm/pkg/avformat"
	"gosm/pkg/protocol/amf"
)

// AVReadCloser .
type AVReadCloser interface {
	ReadAVPacket() (*avformat.AVPacket, error)
	Close() error
}

// Publisher .
type Publisher struct {
	info  *PublisherInfo
	cache *AVCache
	rc    AVReadCloser
}

// PublisherInfo .
type PublisherInfo struct {
	AppName     string
	StreamName  string
	StreamType  string
	PublishTime time.Time
	MetaData    *avformat.MetaData
}

// return onMetaData av packet encoded by amf0
func (p *Publisher) metadata() (*avformat.AVPacket, error) {
	amf := &amf.AMF0{}
	buf := new(bytes.Buffer)

	// convert metadata from struct to map
	metadata, err := json.Marshal(p.info.MetaData)
	if err != nil {
		return nil, err
	}
	mapMetadata := make(map[string]interface{})
	json.Unmarshal(metadata, &mapMetadata)

	// convert from map to amf0
	_, err = amf.WriteString(buf, "onMetaData")
	if err != nil {
		return nil, err
	}
	_, err = amf.WriteTo(buf, &mapMetadata)
	if err != nil {
		return nil, err
	}

	// pack metadata packet
	packet := &avformat.AVPacket{
		TypeID:    avformat.TypeMetadataAMF0,
		Length:    uint32(buf.Len()),
		Timestamp: 0,
		StreamID:  1,
		Body:      buf.Bytes(),
	}
	return packet, nil
}

// parse rtmp onMetadata() data amf packet
func (p *Publisher) parseMetadata(packet *avformat.AVPacket) error {
	amf := &amf.AMF0{}
	r := bytes.NewReader(packet.Body)
	metadata := &avformat.MetaData{
		Server: "github.com/snake-ch/go-streaming-media",
	}

	// @setDataFrame
	if val, err := amf.ReadFrom(r); err != nil {
		return err
	} else if val.(string) != "@setDataFrame" {
		return fmt.Errorf("Publisher: data amf0 '@setDataFrame' missing")
	}

	// onMetaData
	if val, err := amf.ReadFrom(r); err != nil {
		return err
	} else if val.(string) != "onMetaData" {
		return fmt.Errorf("Publisher: data amf0 'onMetaData' missing")
	}

	// audio/video property
	var objects []interface{}
	for r.Len() > 0 {
		property, err := amf.ReadFrom(r)
		if err != nil {
			return err
		}
		objects = append(objects, property)
	}
	if properties, ok := objects[0].(map[string]interface{}); ok {
		if duration, ok := properties["duration"]; ok {
			metadata.Duration = int(duration.(float64))
		}
		if fileSize, ok := properties["fileSize"]; ok {
			metadata.FileSize = int(fileSize.(float64))
		}
		if width, ok := properties["width"]; ok {
			metadata.Width = int(width.(float64))
		}
		if height, ok := properties["height"]; ok {
			metadata.Height = int(height.(float64))
		}
		// video codec id: obs->string, ffmpeg->number
		if videoCodecID, ok := properties["videocodecid"]; ok {
			if _, ok := videoCodecID.(string); ok {
				metadata.VideoCodecID = videoCodecID.(string)
			}
			if _, ok := videoCodecID.(float64); ok {
				metadata.VideoCodecID = videoCodecID.(float64)
			}
		}
		if videoDataRate, ok := properties["videodatarate"]; ok {
			metadata.VideoDataRate = int(videoDataRate.(float64))
		}
		if frameRate, ok := properties["framerate"]; ok {
			metadata.FrameRate = int(frameRate.(float64))
		}
		// audio codec id: obs->string, ffmpeg->number
		if audioCodecID, ok := properties["audiocodecid"]; ok {
			if _, ok := audioCodecID.(string); ok {
				metadata.AudioCodecID = audioCodecID.(string)
			}
			if _, ok := audioCodecID.(float64); ok {
				metadata.AudioCodecID = audioCodecID.(float64)
			}
		}
		if audioSampleRate, ok := properties["audiosamplerate"]; ok {
			metadata.AudioSampleRate = int(audioSampleRate.(float64))
		}
		if audioSampleSize, ok := properties["audiosamplesize"]; ok {
			metadata.AudioSampleSize = int(audioSampleSize.(float64))
		}
		if audioChannels, ok := properties["audiochannels"]; ok {
			metadata.AudioChannels = int(audioChannels.(float64))
		}
		if stereo, ok := properties["stereo"]; ok {
			metadata.Stereo = stereo.(bool)
		}
	}
	p.info.MetaData = metadata
	return nil
}

func (p *Publisher) Close() error {
	return p.rc.Close()
}
