package avformat

import "gosm/pkg/avformat/flv"

// AV packet type
const (
	TypeAudio    = uint8(8)
	TypeVideo    = uint8(9)
	TypeMetadata = uint8(18)
)

// MetaData common metadata, imitates rtmp metadata, see video_file_format_spec_v10.pdf, setion onMetaData
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

// AVPacket common packet carries audio/video/metadata, imitates RTMP message
type AVPacket struct {
	TypeID    uint8
	Length    uint32
	Timestamp uint32
	StreamID  uint32
	Body      []byte
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

// IsHEVCKeyframe .
func (packet *AVPacket) IsHEVCKeyframe() bool {
	body := packet.Body
	return body[0] == flv.AVCKeyFrame<<4|flv.CodevIDHEVC && body[1] == flv.AVCNALU
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
