package live

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gosm/pkg/avformat"
	"gosm/pkg/avformat/flv"
	"gosm/pkg/log"
	"gosm/pkg/protocol/amf"
	"gosm/pkg/protocol/rtmp"
)

// AVReadCloser for publisher
type AVReadCloser interface {
	ReadAVPacket() (*avformat.AVPacket, error)
	Close() error
}

// Publisher .
type Publisher struct {
	ctx    context.Context
	cancel context.CancelFunc
	sps    []byte
	pps    []byte
	info   *PublisherInfo
	cache  *AVCache
	reader AVReadCloser
}

// PublisherInfo .
type PublisherInfo struct {
	AppName     string
	StreamName  string
	StreamType  string
	PublishTime time.Time
	MetaData    *avformat.MetaData
}

// NewPublisher .
func NewPublisher(stream *rtmp.NetStream) (*Publisher, error) {
	pubInfo, ok := stream.Info().(*rtmp.PublishInfo)
	if !ok {
		log.Error("Publisher: information TYPE error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	publisher := &Publisher{
		ctx:    ctx,
		cancel: cancel,
		sps:    nil,
		pps:    nil,
		info: &PublisherInfo{
			AppName:     stream.ConnInfo().App,
			StreamName:  pubInfo.Name,
			StreamType:  pubInfo.Type,
			PublishTime: time.Now(),
			MetaData:    nil,
		},
		cache:  NewAVCache(1),
		reader: stream,
	}
	return publisher, nil
}

// TODO: optimize
// return onMetaData av packet encoded by amf0 format
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

	// convert map to amf0
	_, err = amf.WriteString(buf, "onMetaData")
	if err != nil {
		return nil, err
	}
	_, err = amf.WriteTo(buf, &mapMetadata)
	if err != nil {
		return nil, err
	}

	// pack amf0 to av packet
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
	rd := bytes.NewReader(packet.Body)
	metadata := &avformat.MetaData{
		Server: "github.com/snake-ch/go-streaming-media",
	}

	// @setDataFrame
	if val, err := amf.ReadFrom(rd); err != nil {
		return err
	} else if "@setDataFrame" != val.(string) {
		return fmt.Errorf("Publisher: data amf0 '@setDataFrame' missing")
	}

	// onMetaData
	if val, err := amf.ReadFrom(rd); err != nil {
		return err
	} else if "onMetaData" != val.(string) {
		return fmt.Errorf("Publisher: data amf0 'onMetaData' missing")
	}

	// audio/video property
	var objects []interface{}
	for rd.Len() > 0 {
		property, err := amf.ReadFrom(rd)
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

// parse sps/pps from video seqence header
func (p *Publisher) parseSpsPps(packet *avformat.AVPacket) error {
	if packet == nil {
		packet = p.cache.videoConfig
	}
	if packet == nil {
		return fmt.Errorf("Publisher: video seqence header packet not present")
	}

	seqHeader, err := flv.ParseVideoSeqHeader(packet.Body)
	if err != nil {
		return err
	}
	copy(p.sps, seqHeader.Sps)
	copy(p.pps, seqHeader.Pps)
	return nil
}

func (p *Publisher) close() error {
	p.cancel()
	return p.reader.Close()
}
