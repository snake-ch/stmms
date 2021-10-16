package rtp

import (
	"fmt"
	"gosm/pkg/avformat"
	"gosm/pkg/avformat/avc"
	"gosm/pkg/avformat/flv"
	"time"
)

type RTMPPacker struct{}

func NewRTMPRepacker() *RTMPPacker {
	return &RTMPPacker{}
}

func (packer *RTMPPacker) VideoSeqHdrPacket(sps, pps []byte) (*avformat.AVPacket, error) {
	if sps == nil || pps == nil {
		return nil, fmt.Errorf("RTP: pack rtmp video sequence packet error, SPS or PPS is empty")
	}

	cfg := avc.AVCDecoderConfigurationRecord{
		ConfigurationVersion: 0x01,
		AvcProfileIndication: sps[1],
		ProfileCompatibility: sps[2],
		AvcLevelIndication:   sps[3],
		Sps:                  sps,
		Pps:                  pps,
	}
	avcSeqHdr := &flv.VideoTagData{
		FrameType:       flv.AVCKeyFrame,
		CodecID:         flv.CodevIDAVC,
		AVCPacketType:   flv.AVCSeqHeader,
		CompositionTime: 0,
		Data:            cfg.Bytes(),
	}
	avPayload := avcSeqHdr.Bytes()
	avPacket := &avformat.AVPacket{
		TypeID:    avformat.TypeVideo,
		Length:    uint32(len(avPayload)),
		Timestamp: 0,
		StreamID:  1,
		Body:      avPayload,
	}
	return avPacket, nil
}

// @keyframe	IDR frame or not
// @ts				rtmp packet timestamp
// @pts				rtp timestamp
// @dts				rtp sorted timestamp
// @payload		nalus, avcC format: nalu-size + nalu
func (packer *RTMPPacker) PackVideo(ts uint32, pts, dts int32, payload []byte) (*avformat.AVPacket, error) {
	avcNalu := &flv.VideoTagData{
		FrameType:       flv.AVCInterFrame,
		CodecID:         flv.CodevIDAVC,
		AVCPacketType:   flv.AVCNALU,
		CompositionTime: int32(pts - dts),
		Data:            payload,
	}

	// key frame
	naluType := payload[4] & 0x1F
	if naluType == avc.NALUIDRPicture {
		avcNalu.FrameType = flv.AVCKeyFrame
	}

	avPayload := avcNalu.Bytes()
	avPacket := &avformat.AVPacket{
		TypeID:    avformat.TypeVideo,
		Length:    uint32(len(avPayload)),
		Timestamp: ts,
		StreamID:  1,
		Body:      avPayload,
	}
	return avPacket, nil
}

// copy from obs fixed {0x11, 0x90, 0x56, 0xe5, 0x00}
// defines see https://wiki.multimedia.cx/index.php?title=MPEG-4_Audio
//
// ----------------------------------------------
// AudioObjectType 					[ 5b]	b'00010		AAC-LC
// SamplingFrequencyIndex		[ 4b]	b'0011		48_000
// ChannelConfiguration			[ 4b] b'0010		2 channels: front-left, front-right
// FrameLengthFlag					[ 1b]	0
// DependsOnCoreCoder				[ 1b]	0
// ExtensionFlag						[ 1b]	0
// SyncExtensionType				[11b]	0x2b7
// ExtensionAudioType				[ 5b]	0
// SbrPresentFlag						[ 1b]	0
// ----------------------------------------------

func (packer *RTMPPacker) AudioSeqHdrPacket() *avformat.AVPacket {
	audioSeqHdr := &flv.AudioTagData{
		SoundFormat:    flv.SoundFormatAAC,
		SoundRate:      flv.SoundRate44Khz,
		SoundSize:      flv.SoundRate16Bit,
		SoundType:      flv.SoundTypeStereo,
		AACPackageType: flv.AACSeqHeader,
		Data:           []byte{0x11, 0x90, 0x56, 0xe5, 0x00},
	}
	avPayload := audioSeqHdr.Bytes()
	avPacket := &avformat.AVPacket{
		TypeID:    avformat.TypeAudio,
		Length:    uint32(len(avPayload)),
		Timestamp: uint32(time.Now().Unix() / 1e6),
		StreamID:  1,
		Body:      avPayload,
	}
	return avPacket
}

// TODO: more format, only MPEG4-GENERIC/44100/2 for now
// @ts				rtmp packet timestamp
// @payload 	aac raw
func (packer *RTMPPacker) PackAudio(ts uint32, payload []byte) *avformat.AVPacket {
	audioRaw := &flv.AudioTagData{
		SoundFormat:    flv.SoundFormatAAC,
		SoundRate:      flv.SoundRate44Khz,
		SoundSize:      flv.SoundRate16Bit,
		SoundType:      flv.SoundTypeStereo,
		AACPackageType: flv.AACRaw,
		Data:           payload,
	}
	avPayload := audioRaw.Bytes()
	avPacket := &avformat.AVPacket{
		TypeID:    avformat.TypeAudio,
		Length:    uint32(len(avPayload)),
		Timestamp: ts,
		StreamID:  1,
		Body:      avPayload,
	}
	return avPacket
}
