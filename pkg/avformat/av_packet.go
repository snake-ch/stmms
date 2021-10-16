package avformat

import (
	"fmt"
	"gosm/pkg/avformat/flv"
)

// AV packet type
const (
	TypeAudio        = uint8(8)
	TypeVideo        = uint8(9)
	TypeMetadataAMF0 = uint8(18)
	TypeMetadataAMF3 = uint8(15)
)

// MetaData common metadata, same as rtmp metadata, see video_file_format_spec_v10.pdf, setion onMetaData
type MetaData struct {
	Server          string      `json:"server,omitempty"`
	Duration        int         `json:"duration,omitempty"`
	FileSize        int         `json:"fileSize,omitempty"`
	Width           int         `json:"width,omitempty"`
	Height          int         `json:"height,omitempty"`
	VideoCodecID    interface{} `json:"videocodecid,omitempty"`
	VideoDataRate   int         `json:"videodatarate,omitempty"`
	FrameRate       int         `json:"framerate,omitempty"`
	AudioCodecID    interface{} `json:"audiocodecid,omitempty"`
	AudioSampleRate int         `json:"audiosamplerate,omitempty"`
	AudioSampleSize int         `json:"audiosamplesize,omitempty"`
	AudioChannels   int         `json:"audiochannels,omitempty"`
	Stereo          bool        `json:"stereo,omitempty"`
}

// AVPacket common packet carries audio/video/metadata, same as RTMP message
type AVPacket struct {
	TypeID    uint8
	Length    uint32
	Timestamp uint32
	StreamID  uint32
	Body      []byte
}

// IsVideo .
func (packet *AVPacket) IsVideo() bool {
	return packet.TypeID == TypeVideo
}

// IsAudio .
func (packet *AVPacket) IsAudio() bool {
	return packet.TypeID == TypeAudio
}

// IsAVC .
func (packet *AVPacket) IsAVC() bool {
	body := packet.Body
	return packet.TypeID == TypeVideo && body[0]&0x0F == flv.CodevIDAVC
}

// IsHEVC .
func (packet *AVPacket) IsHEVC() bool {
	body := packet.Body
	return packet.TypeID == TypeVideo && body[0]&0x0F == flv.CodevIDHEVC
}

// IsAVCSeqHeader .
func (packet *AVPacket) IsAVCSeqHeader() bool {
	body := packet.Body
	return body[0] == flv.AVCKeyFrame<<4|flv.CodevIDAVC && body[1] == flv.AVCSeqHeader
}

// IsHEVCSeqHeader .
func (packet *AVPacket) IsHEVCSeqHeader() bool {
	body := packet.Body
	return body[0] == flv.AVCKeyFrame<<4|flv.CodevIDHEVC && body[1] == flv.AVCSeqHeader
}

// IsAVCKeyframe .
func (packet *AVPacket) IsAVCKeyframe() bool {
	body := packet.Body
	return body[0] == flv.AVCKeyFrame<<4|flv.CodevIDAVC && body[1] == flv.AVCNALU
}

// IsAVCInterframe .
func (packet *AVPacket) IsAVCInterframe() bool {
	body := packet.Body
	return body[0] == flv.AVCInterFrame<<4|flv.CodevIDAVC && body[1] == flv.AVCNALU
}

// IsHEVCKeyframe .
func (packet *AVPacket) IsHEVCKeyframe() bool {
	body := packet.Body
	return body[0] == flv.AVCKeyFrame<<4|flv.CodevIDHEVC && body[1] == flv.AVCNALU
}

// IsHEVCInterframe .
func (packet *AVPacket) IsHEVCInterframe() bool {
	body := packet.Body
	return body[0] == flv.AVCInterFrame<<4|flv.CodevIDHEVC && body[1] == flv.AVCNALU
}

// IsAAC .
func (packet *AVPacket) IsAAC() bool {
	body := packet.Body
	return body[0]>>4 == flv.SoundFormatAAC
}

// IsAACSeqHeader .
func (packet *AVPacket) IsAACSeqHeader() bool {
	body := packet.Body
	return body[0]>>4 == flv.SoundFormatAAC && body[1] == flv.AACSeqHeader
}

// IsAACRaw .
func (packet *AVPacket) IsAACRaw() bool {
	body := packet.Body
	return body[0]>>4 == flv.SoundFormatAAC && body[1] == flv.AACRaw
}

// String .
func (packet *AVPacket) String() string {
	return fmt.Sprintf("{TypeID: %d, Length: %d, TimeStramp: %d, StreamID: %d}",
		packet.TypeID, packet.Length, packet.Timestamp, packet.StreamID)
}
