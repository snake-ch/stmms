package flv

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// tag type
const (
	TagTypeAudio    = uint8(8)
	TagTypeVideo    = uint8(9)
	TagTypeMetadata = uint8(18)
)

// audio
const (
	// SoundFormat: UB[4]
	// 0 = Linear PCM, platform endian
	// 1 = ADPCM
	// 2 = MP3
	// 3 = Linear PCM, little endian
	// 4 = Nellymoser 16-kHz mono
	// 5 = Nellymoser 8-kHz mono
	// 6 = Nellymoser
	// 7 = G.711 A-law logarithmic PCM
	// 8 = G.711 mu-law logarithmic PCM
	// 9 = reserved
	// 10 = AAC
	// 11 = Speex
	// 14 = MP3 8-Khz
	// 15 = Device-specific sound
	// Formats 7, 8, 14, and 15 are reserved for internal use
	// AAC is supported in Flash Player 9,0,115,0 and higher.
	// Speex is supported in Flash Player 10 and higher.
	SoundFormatAAC = uint8(10)

	// SoundRate: UB[2]
	// Sampling rate
	// 0 = 5.5-kHz For AAC: always 3
	// 1 = 11-kHz
	// 2 = 22-kHz
	// 3 = 44-kHz
	SoundRate5_5Khz = uint8(0)
	SoundRate11Khz  = uint8(1)
	SoundRateKhz    = uint8(2)
	SoundRate44Khz  = uint8(3)

	// SoundSize: UB[1]
	// 0 = snd8Bit
	// 1 = snd16Bit
	// Size of each sample.
	// This parameter only pertains to uncompressed formats.
	// Compressed formats always decode to 16 bits internally
	SoundRate8Bit  = uint8(0)
	SoundRate16Bit = uint8(1)

	// SoundType: UB[1]
	// 0 = sndMono
	// 1 = sndStereo
	// Mono or stereo sound For Nellymoser: always 0
	// For AAC: always 1
	SoundTypeMono   = uint8(0)
	SoundTypeStereo = uint8(1)

	// 0: AAC sequence header
	// 1: AAC raw
	AACSeqHeader = uint8(0)
	AACRaw       = uint8(1)
)

// video
const (
	// 1: keyframe (for AVC, a seekable frame)
	// 2: inter frame (for AVC, a non- seekable frame)
	// 3: disposable inter frame (H.263 only)
	// 4: generated keyframe (reserved for server use only)
	// 5: video info/command frame
	AVCKeyFrame   = uint8(1)
	AVCInterFrame = uint8(2)

	// 1: JPEG (currently unused)
	// 2: Sorenson H.263
	// 3: Screen video
	// 4: On2 VP6
	// 5: On2 VP6 with alpha channel
	// 6: Screen video version 2
	// 7: AVC/H.264
	// 12: HEVC/H.265
	CodevIDAVC  = uint8(7)
	CodevIDHEVC = uint8(12)

	// 0: AVC sequence header
	// 1: AVC NALU
	// 2: AVC end of sequence
	AVCSeqHeader = uint8(0)
	AVCNALU      = uint8(1)
	AVCEndOfSeq  = uint8(2)
)

//
var (
	SoundFormat = map[byte]string{
		0:  "Linear PCM, platform endian",
		1:  "ADPCM",
		2:  "MP3",
		3:  "Linear PCM, little endian",
		4:  "Nellymoser 16kHz mono",
		5:  "Nellymoser 8kHz mono",
		6:  "Nellymoser",
		7:  "G.711 A-law logarithmic PCM",
		8:  "G.711 mu-law logarithmic PCM",
		9:  "reserved",
		10: "AAC",
		11: "Speex",
		14: "MP3 8Khz",
		15: "Device-specific sound"}

	SoundRate = map[byte]int{
		0: 5500,
		1: 11000,
		2: 22000,
		3: 44000,
	}

	SoundSize = map[byte]string{
		0: "8Bit",
		1: "16Bit",
	}

	SoundType = map[byte]string{
		0: "Mono",
		1: "Stereo",
	}

	FrameType = map[byte]string{
		1: "keyframe (for AVC, a seekable frame)",
		2: "inter frame (for AVC, a non-seekable frame)",
		3: "disposable inter frame (H.263 only)",
		4: "generated keyframe (reserved for server use only)",
		5: "video info/command frame",
	}

	CodecID = map[byte]string{
		1:  "JPEG (currently unused)",
		2:  "Sorenson H.263",
		3:  "Screen video",
		4:  "On2 VP6",
		5:  "On2 VP6 with alpha channel",
		6:  "Screen video version 2",
		7:  "AVC/H.264",
		12: "HEVC/H.265",
	}
)

// FlvHeader .
var FlvHeader = []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09}

// Tag FlvTag.
type Tag struct {
	TagHeader *TagHeader
	TagData   []byte
}

// TagHeader .
type TagHeader struct {
	TagType   uint8
	DataSize  uint32
	Timestamp uint32
	StreamID  uint32 // always 0
}

// AudioTagData .
type AudioTagData struct {
	SoundFormat    uint8
	SoundRate      uint8
	SoundSize      uint8
	SoundType      uint8
	AACPackageType uint8
	Data           []byte
}

// Bytes .
func (tag *AudioTagData) Bytes() []byte {
	buf := bytes.NewBuffer([]byte{})
	buf.WriteByte(tag.SoundFormat<<4 | tag.SoundRate<<2 | tag.SoundSize<<1 | tag.SoundType)
	buf.WriteByte(tag.AACPackageType)
	buf.Write(tag.Data)
	return buf.Bytes()
}

// VideoTagData .
type VideoTagData struct {
	FrameType       uint8
	CodecID         uint8
	AVCPacketType   uint8
	CompositionTime int32
	Data            []byte
}

// Bytes .
func (tag *VideoTagData) Bytes() []byte {
	buf := new(bytes.Buffer)
	buf.WriteByte(tag.FrameType<<4 | tag.CodecID)
	buf.WriteByte(tag.AVCPacketType)
	buf.WriteByte(byte(tag.CompositionTime))
	buf.WriteByte(byte(tag.CompositionTime >> 8))
	buf.WriteByte(byte(tag.CompositionTime >> 16))
	buf.Write(tag.Data)
	return buf.Bytes()
}

// IsMetadata .
func (tag *Tag) IsMetadata() bool {
	return tag.TagHeader.TagType == TagTypeMetadata
}

// IsAAC .
func (tag *Tag) IsAAC() bool {
	return tag.TagHeader.TagType == TagTypeAudio && tag.TagData[0]>>4 == SoundFormatAAC
}

// IsAVC .
func (tag *Tag) IsAVC() bool {
	return tag.TagHeader.TagType == TagTypeVideo && tag.TagData[0]&0x0F == CodevIDAVC
}

// IsHEVC .
func (tag *Tag) IsHEVC() bool {
	return tag.TagHeader.TagType == TagTypeVideo && tag.TagData[0]&0x0F == CodevIDHEVC
}

// ParseTagHeader .
func (tag *Tag) ParseTagHeader(p []byte) error {
	if len(p) < 11 {
		return fmt.Errorf("FLV: invalid to parse audio tag data, length = %d", len(p))
	}
	tag.TagHeader = &TagHeader{}
	tag.TagHeader.TagType = p[0]
	tag.TagHeader.DataSize = binary.BigEndian.Uint32(p[1:4])
	tag.TagHeader.Timestamp = uint32(p[7])<<24 + binary.BigEndian.Uint32(p[4:7])
	tag.TagHeader.StreamID = 0 // always 0
	return nil
}

// ParseAACAudioData .
func ParseAACAudioData(p []byte) (*AudioTagData, error) {
	if len(p) < 2 {
		return nil, fmt.Errorf("FLV: invalid length to parse aac audio data, len=%d", len(p))
	}

	tagData := &AudioTagData{}
	tagData.SoundFormat = p[0] >> 4
	tagData.SoundRate = (p[0] >> 2) & 0x3
	tagData.SoundSize = (p[0] >> 1) & 0x1
	tagData.SoundType = p[0] & 0x1

	// aac
	if tagData.SoundFormat != SoundFormatAAC {
		return nil, fmt.Errorf("FLV: invalid aac audio sound format")
	}
	tagData.AACPackageType = p[1]
	tagData.Data = p[2:]

	return tagData, nil
}

// ParseAVCVideoPackage .
func ParseAVCVideoPackage(p []byte) (*VideoTagData, error) {
	if len(p) < 6 {
		return nil, fmt.Errorf("FLV: invalid length to parse avc video data, len=%d", len(p))
	}

	tagData := &VideoTagData{}
	tagData.FrameType = p[0] >> 4
	tagData.CodecID = p[0] & 0x0F

	// avc
	if tagData.FrameType != AVCKeyFrame && tagData.FrameType != AVCInterFrame {
		return nil, fmt.Errorf("FLV: invalid avc video frame type")
	}
	tagData.AVCPacketType = p[1]
	for idx := 0; idx < 3; idx++ {
		tagData.CompositionTime = tagData.CompositionTime<<8 + int32(p[2+idx])
	}
	tagData.Data = p[5:]

	return tagData, nil
}
