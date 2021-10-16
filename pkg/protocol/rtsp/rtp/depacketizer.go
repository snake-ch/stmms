package rtp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"gosm/pkg/avformat/avc"
	"gosm/pkg/log"
	"io"
)

type Depacketizer struct {
	fragments []*Packet // FU-* or audio packet cache
	sps       []byte
	pps       []byte
	sei       []byte
	nalus     *bytes.Buffer
}

func NewDepacketizer() *Depacketizer {
	return &Depacketizer{
		fragments: make([]*Packet, 0),
		sps:       nil,
		pps:       nil,
		sei:       nil,
		nalus:     bytes.NewBuffer([]byte{}),
	}
}

func (depacketizer *Depacketizer) SPS() []byte {
	return depacketizer.sps
}

func (depacketizer *Depacketizer) PPS() []byte {
	return depacketizer.pps
}

func (depacketizer *Depacketizer) Nalus() []byte {
	nalus := make([]byte, depacketizer.nalus.Len())
	io.ReadFull(depacketizer.nalus, nalus)
	return nalus
}

func (depacketizer *Depacketizer) DepacketizeVideo(packet *Packet) (bool, error) {
	// ---------------------------------------------------------
	//	H.264/AVC
	// ---------------------------------------------------------
	marker := packet.header.m == 0x01 // video full frame boundary
	fuIndicator := packet.payload[0]
	naluType := fuIndicator & 0x1F

	// single nal unit
	if naluType > avc.NALUUnspecified && naluType <= avc.NALUReserved5 {
		if err := depacketizer.parseNalu(packet.payload); err != nil {
			return marker, err
		}
	}

	// rtp nalu payload
	switch naluType {
	case NALUSTAPA:
		payload := packet.payload[1:] // skip 1-byte STAP-A header
		if err := depacketizer.parseSTAP(payload); err != nil {
			return marker, err
		}
	case NALUSTAPB:
		log.Debug("RTP: not support STAP-B")
	case NALUMTAP16:
		log.Debug("RTP: not support MTAP-16")
	case NALUMTAP24:
		log.Debug("RTP: not support MTAP-24")
	case NALUFUA:
		fuHeader := packet.payload[1]
		if (fuHeader>>7)&0x01 == 0x01 { // FU-A start
			depacketizer.fragments = depacketizer.fragments[:0]
		}
		depacketizer.fragments = append(depacketizer.fragments, packet) // FU-A continue
		if (fuHeader>>6)&0x01 != 0x01 {                                 // FU-A end
			return marker, nil
		}

		nalu := make([]byte, 0)
		nalu = append(nalu, fuIndicator&0x60|fuHeader&0x1F) // restore nalu-header
		for _, fragment := range depacketizer.fragments {   // restore nalu-rbsp
			nalu = append(nalu, fragment.payload[2:]...)
		}
		if err := depacketizer.parseNalu(nalu); err != nil {
			return marker, err
		}
		return marker, nil
	case NALUFUB:
		log.Debug("RTP: not support FU-B")
	}

	// ---------------------------------------------------------
	// TODO: support H.265/HEVC
	// ---------------------------------------------------------
	return marker, nil
}

// parse sps / pps / sei from STAP-*
func (depacketizer *Depacketizer) parseSTAP(nalus []byte) error {
	for pos := 0; pos != len(nalus); {
		if pos > len(nalus) {
			return fmt.Errorf("RTP Parser: parse STAP error, pos:%d, out of range:%d", pos, len(nalus))
		}

		lenOfNalu := int(binary.BigEndian.Uint16(nalus[pos:]))
		pos = pos + 2

		switch nalus[pos] & 0x1F {
		case avc.NALUSEI:
			depacketizer.sei = nalus[pos : pos+lenOfNalu]
		case avc.NALUSPS:
			depacketizer.sps = nalus[pos : pos+lenOfNalu]
		case avc.NALUPPS:
			depacketizer.pps = nalus[pos : pos+lenOfNalu]
		case avc.NALUNonIDRPicture:
			fallthrough
		case avc.NALUIDRPicture: // avcC format: nalu-size + nalu
			if err := binary.Write(depacketizer.nalus, binary.BigEndian, uint32(lenOfNalu)); err != nil {
				return err
			}
			if _, err := depacketizer.nalus.Write(nalus[pos : pos+lenOfNalu]); err != nil {
				return nil
			}
		}
		pos = pos + lenOfNalu
	}
	return nil
}

// TODO: refresh outbound sps & pps if update
func (depacketizer *Depacketizer) parseNalu(nalu []byte) error {
	// sps & pps & sei
	naluType := nalu[0] & 0x1F
	switch naluType {
	case avc.NALUSEI:
		depacketizer.sei = nalu
	case avc.NALUSPS:
		depacketizer.sps = nalu
	case avc.NALUPPS:
		depacketizer.pps = nalu
	}

	// avcC format: nalu-size + nalu
	if err := binary.Write(depacketizer.nalus, binary.BigEndian, uint32(len(nalu))); err != nil {
		return err
	}
	if _, err := depacketizer.nalus.Write(nalu); err != nil {
		return err
	}
	return nil
}

// rfc3640 2.11.  Global Structure of Payload Format
//
// +---------+-----------+-----------+---------------+
// | RTP     | AU Header | Auxiliary | Access Unit   |
// | Header  | Section   | Section   | Data Section  |
// +---------+-----------+-----------+---------------+
//
//           <----------RTP Packet Payload----------->
//
// rfc3640 3.2.1.  The AU Header Section
//
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+- .. -+-+-+-+-+-+-+-+-+-+
// |AU-headers-length|AU-header|AU-header|      |AU-header|padding|
// |                 |   (1)   |   (2)   |      |   (n)   | bits  |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+- .. -+-+-+-+-+-+-+-+-+-+
//
// rfc3640 3.3.6.  High Bit-rate AAC

type AudioFrame struct {
	Timestamp uint32 // same as rtp timestamp
	Raw       []byte // samples of one aac frame
}

func (depacketizer *Depacketizer) DepacketizeAudio(packet *Packet) ([]*AudioFrame, error) {
	// aac raw payload
	depacketizer.fragments = append(depacketizer.fragments, packet)
	if packet.header.m != 0x01 {
		return nil, nil
	}

	// copy rtp payload and reset cache
	rtpPayload := make([]byte, 0)
	for _, fragment := range depacketizer.fragments {
		rtpPayload = append(rtpPayload, fragment.payload...)
	}
	depacketizer.fragments = depacketizer.fragments[:0]

	auHeadersLength := uint16(rtpPayload[0])<<8 | uint16(rtpPayload[1])
	numOfAuHeaders := auHeadersLength / 16
	audioFrames := make([]*AudioFrame, numOfAuHeaders)
	for pos, idx := uint16(2), uint16(0); idx < numOfAuHeaders; idx++ {
		aacDataLen := uint16(rtpPayload[pos])<<5 | uint16(rtpPayload[pos+1])>>3
		pos += 2
		aacPayload := rtpPayload[pos : pos+aacDataLen]
		pos += aacDataLen

		audioFrame := &AudioFrame{
			Timestamp: uint32(idx*1024) + packet.header.ts, // 1024 samples per aac frame
			Raw:       aacPayload,
		}
		audioFrames[numOfAuHeaders] = audioFrame
	}
	return audioFrames, nil
}
