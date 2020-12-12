package flv

import (
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
		3: 44000}

	SoundSize = map[byte]string{
		0: "8Bit",
		1: "16Bit"}

	SoundType = map[byte]string{
		0: "Mono",
		1: "Stereo"}

	FrameType = map[byte]string{
		1: "keyframe (for AVC, a seekable frame)",
		2: "inter frame (for AVC, a non-seekable frame)",
		3: "disposable inter frame (H.263 only)",
		4: "generated keyframe (reserved for server use only)",
		5: "video info/command frame"}

	CodecID = map[byte]string{
		1:  "JPEG (currently unused)",
		2:  "Sorenson H.263",
		3:  "Screen video",
		4:  "On2 VP6",
		5:  "On2 VP6 with alpha channel",
		6:  "Screen video version 2",
		7:  "AVC/H.264",
		12: "HEVC/H.265"}
)

// FlvHeader .
var FlvHeader = []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09}

// Tag FlvTag.
type Tag struct {
	tagHeader *TagHeader
	tagData   []byte
}

// TagHeader .
type TagHeader struct {
	tagType   uint8
	dataSize  uint32
	timestamp uint32
	streamID  uint32 // always 0
}

// AudioTagData .
type AudioTagData struct {
	soundFormat uint8
	soundRate   uint8
	soundSize   uint8
	soundType   uint8
	soundData   []byte
}

// VideoTagData .
type VideoTagData struct {
	frameType     uint8
	codecID       uint8
	avcPacketType uint8
	videoData     []byte
}

// AudioSeqHeader .
// see ISO 14496-3 setion for the description MP4/F4V file
type AudioSeqHeader struct {
}

// VideoSeqHeader .
// see ISO 14496-15 5_2_4_1 setion for the description of AVCDecoderConfigurationRecord
type VideoSeqHeader struct {
	ConfigurationVersion byte
	AvcProfileIndication byte
	ProfileCompatibility byte
	AvcLevelIndication   byte
	LengthSizeMinusOne   byte
	Sps                  []byte // pictureParameterSetNALUnits
	Pps                  []byte // sequenceParameterSetNALUnits
}

// IsMetadata .
func (tag *Tag) IsMetadata() bool {
	return tag.tagHeader.tagType == TagTypeMetadata
}

// IsAAC .
func (tag *Tag) IsAAC() bool {
	return tag.tagHeader.tagType == TagTypeAudio && tag.tagData[0]>>4 == SoundFormatAAC
}

// IsAVC .
func (tag *Tag) IsAVC() bool {
	return tag.tagHeader.tagType == TagTypeVideo && tag.tagData[0]&0x0F == CodevIDAVC
}

// IsHEVC .
func (tag *Tag) IsHEVC() bool {
	return tag.tagHeader.tagType == TagTypeVideo && tag.tagData[0]&0x0F == CodevIDHEVC
}

// ParseTagHeader .
func (tag *Tag) ParseTagHeader(b []byte) error {
	if len(b) < 11 {
		return fmt.Errorf("invalid to parse audio tag data, length = %d", len(b))
	}
	tag.tagHeader = &TagHeader{}
	tag.tagHeader.tagType = b[0]
	tag.tagHeader.dataSize = binary.BigEndian.Uint32(b[1:4])
	tag.tagHeader.timestamp = uint32(b[7])<<24 + binary.BigEndian.Uint32(b[4:7])
	tag.tagHeader.streamID = 0 // always 0
	return nil
}

// ParseAudioTagData .
func parseAudioTagData(b []byte) (tagData *AudioTagData, err error) {
	if len(b) < 1 {
		err = fmt.Errorf("invalid length to parse audio tag data")
	}
	// audio parameters
	param := b[0]
	tagData.soundFormat = param >> 4
	tagData.soundRate = (param >> 2) & 0x3
	tagData.soundSize = (param >> 1) & 0x1
	tagData.soundType = param & 0x1
	// audio data
	if len(b) > 2 {
		tagData.soundData = b[1:]
	}
	return
}

// ParseVideoTagData .
func parseVideoTagData(b []byte) (tagData *VideoTagData, err error) {
	if len(b) < 1 {
		err = fmt.Errorf("invalid length to parse video tag data")
	}
	// video parameters
	param := b[0]
	tagData.frameType = param >> 4
	tagData.codecID = param & 0x0F
	// video data
	if len(b) > 2 {
		tagData.videoData = b[1:]
	}
	return
}

// ParseVideoSeqHeader parse AVC video packet sequence header from flv video tag data
func ParseVideoSeqHeader(tagData []byte) (*VideoSeqHeader, error) {
	if len(tagData) < 11 {
		return nil, fmt.Errorf("invalid length to parse video tag data")
	}

	seqHeader := &VideoSeqHeader{}
	seqHeader.ConfigurationVersion = tagData[5]
	seqHeader.AvcProfileIndication = tagData[6]
	seqHeader.ProfileCompatibility = tagData[7]
	seqHeader.AvcLevelIndication = tagData[8]
	seqHeader.LengthSizeMinusOne = tagData[9]&0x03 + 1 // NAL-Unit size

	// extract SPS
	var pos uint16 = 10
	numOfSps := int(tagData[pos] & 0x1F) // numOfSequenceParameterSets
	pos = pos + 1
	for idx := 0; idx < numOfSps; idx++ {
		lenOfSps := binary.BigEndian.Uint16(tagData[pos:]) // sequenceParameterSetLength
		pos = pos + 2
		seqHeader.Sps = append(seqHeader.Sps, tagData[pos:pos+lenOfSps]...)
		pos = pos + lenOfSps
	}

	// extract PPS
	numOfPps := int(tagData[pos] & 0x1F) // numOfPictureParameterSets
	pos = pos + 1
	for idx := 0; idx < numOfPps; idx++ {
		lenOfPps := binary.BigEndian.Uint16(tagData[pos:]) // pictureParameterSetLength
		pos = pos + 2
		seqHeader.Pps = append(seqHeader.Pps, tagData[pos:pos+lenOfPps]...)
		pos = pos + lenOfPps
	}

	return seqHeader, nil
}
