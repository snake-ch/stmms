package aac

import (
	"fmt"
	"io"
)

const ADTSHeaderLength = 7

// AACSampleRate AAC sample rate
var AACSampleRate = []int{96000, 88200, 64000, 48000, 44100, 32000, 24000, 22050, 16000, 12000, 11025, 8000, 7350}

type AACParser struct {
	w                   io.Writer
	audioSpecificConfig *AudioSpecificConfig
}

func NewAACParser(w io.Writer) *AACParser {
	return &AACParser{w: w}
}

// ADTS see ISO_IEC_14496-3 setion for the description MP4/F4V file .
type AudioSpecificConfig struct {
	// AudioSpecificConfig
	AudioObjectType        byte
	SamplingFrequencyIndex byte
	ChannelConfiguration   byte

	// GASpecificConfig
	FrameLengthFlag    byte
	DependsOnCoreCoder byte
	ExtensionFlag      byte
}

// AudioSpecificConfig parse AAC audio spcific configuration, only support audio object type=2 (AAC-LC)
func (parser *AACParser) ParseAudioSpecificConfig(p []byte) error {
	if len(p) < 2 {
		return fmt.Errorf("AAC: invalid length to parse aac sequence header, len=%d", len(p))
	}
	cfg := &AudioSpecificConfig{}
	cfg.AudioObjectType = (p[0] & 0xF8) >> 3
	cfg.SamplingFrequencyIndex = (p[0]&0x07)<<1 | p[1]>>7
	cfg.ChannelConfiguration = (p[1] >> 3) & 0x0F
	cfg.FrameLengthFlag = (p[1] >> 2) & 0x01
	cfg.DependsOnCoreCoder = (p[1] >> 1) & 0x01
	cfg.ExtensionFlag = p[1] & 0x01
	parser.audioSpecificConfig = cfg
	return nil
}

// see ISO_IEC_14496-3
//   ----------------------------------------------------
//   syncword                 [12b] 0xFFF
//   ID                       [ 1b] 0=MPEG-4, 1=MPEG-2
//   layer                    [ 2b] 0
//   protection_absent        [ 1b] 0=crc, 1=none
//   profile                  [ 2b] 1=AAC Main 2=AAC LC 3=AAC SSR 4=AAC LTP
//   sampling_frequency_index [ 4b]
//   private_bit              [ 1b] 0
//   channel_configuration    [ 3b]
//   origin_copy              [ 1b] 0
//   home                     [ 1b] 0
//   ------------------------------------
//   copyright_identification_bit   [ 1b] 0
//   copyright_identification_start [ 1b] 0
//   aac_frame_length               [13b]
//   adts_buffer_fullness           [11b] 0x7FF
//   no_raw_data_blocks_in_frame    [ 2b] 0
//
// @param <length> raw aac payload length
func (parser *AACParser) WriteADTS(length uint16) (n int, err error) {
	aacFrameLength := length + ADTSHeaderLength
	buf := make([]byte, ADTSHeaderLength)
	buf[0] = 0xFF                                                    // syncword 0(8)
	buf[1] |= 0x0F << 4                                              // syncword 1(4)
	buf[1] |= 0x00 << 3                                              // ID 1(4)
	buf[1] |= 0x00 << 1                                              // layer 1(4)
	buf[1] |= 0x01                                                   // protection_absent 1(4)
	buf[2] |= (parser.audioSpecificConfig.AudioObjectType - 1) << 6  // profile 2(2)
	buf[2] |= parser.audioSpecificConfig.SamplingFrequencyIndex << 2 // sampling_frequency_index 2(4)
	buf[2] |= 0x00 << 1                                              // private_bit 2(1)
	buf[2] |= parser.audioSpecificConfig.ChannelConfiguration >> 6   // channel_configuration 2(1)
	buf[3] |= parser.audioSpecificConfig.ChannelConfiguration << 6   // channel_configuration 3(2)
	buf[3] |= 0x00 << 5                                              // origin_copy 3(4)
	buf[3] |= 0x00 << 4                                              // home 3(4)
	buf[3] |= 0x00 << 3                                              // copyright_identification_bit 3(4)
	buf[3] |= 0x00 << 2                                              // copyright_identification_start 3(4)
	buf[3] |= byte((aacFrameLength & 0x1800) >> 14)                  // aac_frame_length 3(2)
	buf[4] |= byte((aacFrameLength & 0x7f8) >> 3)                    // aac_frame_length 4(8)
	buf[5] |= byte((aacFrameLength & 0x07) << 5)                     // aac_frame_length 5(3)
	buf[5] |= 0x7FF >> 6                                             // adts_buffer_fullness 5(5)
	buf[6] |= 0x7FF & 0x3F << 2                                      // adts_buffer_fullness 6(6)
	buf[6] |= 0x00                                                   // no_raw_data_blocks_in_frame 6(2)
	return parser.w.Write(buf)
}

func (parser *AACParser) Write(p []byte) (n int, err error) {
	return parser.w.Write(p)
}
