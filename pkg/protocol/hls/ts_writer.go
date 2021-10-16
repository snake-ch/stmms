package hls

import (
	"bufio"
	"bytes"
	"gosm/pkg/avformat"
	"gosm/pkg/avformat/aac"
	"gosm/pkg/avformat/avc"
	"gosm/pkg/avformat/flv"
	"os"
)

// TS:
//   +-+-+-+-+     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   |  TS   |  =  |  Packet 1 |  Packet 2 |  Packet 3 |    ...    | Packet n-1 | Packet n |
//   +-+-+-+-+     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

// Packet:               4 bytes             184 bytes
//   +-+-+-+-+-+    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   |  Packet | =  | Packet header |       Packet data       |
//   +-+-+-+-+-+    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

// PAT/PMT(Packet)
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | ts header |    PAT/PMT    |   Stuffing Bytss  |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

// Video/Audio(Packet)
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | ts header |   adaptation field    |      payload(pes 1)     |--> 1st Packet
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | ts header |              payload(pes 2)                     |--> 2nd Packet
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | ts header |                   ...                           |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | ts header |             payload(pes n-1)                    |--> n-1 Packet
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | ts header |   adaptation field    |      payload(pes n)     |-->   n Packet
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

type TSMuxer struct {
	fp     *os.File
	rw     *bufio.ReadWriter
	packet []byte
}

func NewTSMuxer(fn string) (*TSMuxer, error) {
	fp, err := os.Create(fn)
	if err != nil {
		return nil, err
	}
	muxer := &TSMuxer{
		fp:     fp,
		rw:     bufio.NewReadWriter(bufio.NewReader(fp), bufio.NewWriter(fp)),
		packet: make([]byte, 188),
	}
	// PAT/PMT
	if _, err := muxer.rw.Write(FixedPATPMT); err != nil {
		return nil, err
	}
	return muxer, nil
}

// Reset open next ts fragment file
func (muxer *TSMuxer) Reset(fn string) error {
	if err := muxer.Close(); err != nil {
		return err
	}

	fp, err := os.Create(fn)
	if err != nil {
		return err
	}
	muxer.fp = fp
	muxer.rw.Reader.Reset(fp)
	muxer.rw.Writer.Reset(fp)
	// PAT/PMT
	if _, err := muxer.fp.Write(FixedPATPMT); err != nil {
		return err
	}
	return nil
}

func (muxer *TSMuxer) Close() error {
	if err := muxer.rw.Flush(); err != nil {
		return err
	}
	return muxer.fp.Close()
}

// TODO: optimized
func (muxer *TSMuxer) Write(pid uint16, keyframe bool, cc *uint8, pcr uint64, pes []byte) error {
	tsFirst := true      // first ts packet flag
	tsIdx := 0           // ts packet write position
	pesIdx := 0          // pes packet read position
	lenOfPes := len(pes) // pes packet size

	for pesIdx != lenOfPes {
		*cc = (*cc + 1) & 0x0F

		muxer.packet[0] = 0x47 // sync byte
		muxer.packet[1] = 0x00 // error indicator, unit start indicator, ts priority
		if tsFirst {
			muxer.packet[1] |= 0x40
		}
		muxer.packet[1] |= uint8((pid >> 8) & 0x1F) // [5b] pid high
		muxer.packet[2] = uint8(pid & 0xFF)         // [8b] pid low
		muxer.packet[3] = 0x10 | *cc                // scrambling, adaptation, continuity_counter
		tsIdx += 4

		if tsFirst {
			muxer.packet[3] |= 0x20 // adaptation
			muxer.packet[4] = 7     // adaptation length
			if keyframe {
				muxer.packet[5] = 0x40 // random access
			}
			muxer.packet[5] |= 0x10            // pcr flag
			muxer.packet[6] = uint8(pcr >> 25) // pcr
			muxer.packet[7] = uint8(pcr >> 17)
			muxer.packet[8] = uint8(pcr >> 9)
			muxer.packet[9] = uint8(pcr >> 1)
			muxer.packet[10] = uint8(pcr<<7) | 0x7e
			muxer.packet[11] = 0x00

			tsIdx += 8
			tsFirst = false
		}

		tsRemains := 188 - tsIdx
		pesRemains := lenOfPes - pesIdx

		// 1st ~ n-1 packet
		if tsRemains <= pesRemains {
			copy(muxer.packet[tsIdx:], pes[pesIdx:])
			pesIdx += tsRemains
		} else {
			lenOfStuff := tsRemains - pesRemains

			// last packet is 1st packet, within adaptation field
			if muxer.packet[3]&0x20 != 0 {
				// length of ts header & adaptation
				offset := int(4 + muxer.packet[4])

				// stuff with 0xFF
				muxer.packet[4] += byte(lenOfStuff)
				for idx := 0; idx < lenOfStuff; idx++ {
					muxer.packet[offset+idx] = 0xFF
				}
			}

			// last packet is n-th pakcet, without adaptation field
			if muxer.packet[3]&0x20 == 0 {
				muxer.packet[3] |= 0x20
				muxer.packet[4] = uint8(lenOfStuff - 1)
				if lenOfStuff >= 2 {
					// [8b] adaptation field flag
					muxer.packet[5] = 0x00
					// stuff with 0xFF
					for idx := 0; idx < lenOfStuff-2; idx++ {
						muxer.packet[6+idx] = 0xFF
					}
				}
			}

			// append pes to ts end
			copy(muxer.packet[188-pesRemains:], pes[pesIdx:])
			pesIdx += pesRemains
		}

		tsIdx = 0
		if _, err := muxer.rw.Write(muxer.packet); err != nil {
			return err
		}
	}
	return nil
}

// video:
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | start code(4 byte) | nalu header(1 byte) |      h264 data(x byte)       |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
// audio:
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | adts header(7 byte) |      aac data(x byte)       |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
// ES -> PES:
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | pes header | nalu(0x09) | 1byte | nalu |     | nalu(0x67) |     | nalu(0x68) |     | nalu(0x65) |     |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//                     aud       0xF0                    SPS                PPS                 I
//
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   | pes header | nalu(0x09) | 1byte | nalu |     | nalu(0x41) |     |
//   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//                     aud       0xF0                     P

// Writer wraps pes packetizer and ts muxer
type Writer struct {
	stream string

	// pes
	pesPacket *bytes.Buffer
	avcParser *avc.AVCParser
	aacParser *aac.AACParser

	// m3u8
	m3u8 *M3U8

	// ts
	tsMuxer *TSMuxer
	audioCC uint8
	videoCC uint8
}

func NewWriter(stream string) (w *Writer, err error) {
	w = &Writer{}
	w.stream = stream
	w.pesPacket = &bytes.Buffer{}
	w.avcParser = avc.NewAVCParser(w.pesPacket)
	w.aacParser = aac.NewAACParser(w.pesPacket)
	w.m3u8 = NewM3U8(stream)
	w.tsMuxer, err = NewTSMuxer(w.m3u8.NextSegment())
	w.audioCC = 0
	w.videoCC = 0

	return w, err
}

// Write
func (w *Writer) Write(packet *avformat.AVPacket) error {
	if packet.IsVideo() {
		return w.packVideoPES(packet)
	}
	if packet.IsAudio() {
		return w.packAudioPES(packet)
	}
	return nil
}

// TODO: support HEVC
func (w *Writer) packVideoPES(packet *avformat.AVPacket) error {
	videoTag, err := flv.ParseAVCVideoPackage(packet.Body)
	if err != nil {
		return err
	}

	// cache video extradata
	if packet.IsAVCSeqHeader() {
		if err := w.avcParser.ParseExtradata(videoTag.Data); err != nil {
			return err
		}
		return nil
	}

	// pes header
	dts := uint64(packet.Timestamp) * 90
	h := w.parseVideoPESHeader(videoTag, dts)
	if _, err := h.WriteTo(w.pesPacket); err != nil {
		return err
	}
	// pes payload
	if packet.IsAVC() {
		if err := w.avcParser.WriteAnnexB(videoTag.Data); err != nil {
			return err
		}
	}

	// propagate to ts muxer
	keyframe := packet.IsAVCKeyframe() || packet.IsHEVCKeyframe()
	w.tsMuxer.Write(PIDVideo, keyframe, &w.videoCC, dts, w.pesPacket.Bytes())
	w.pesPacket.Reset()

	// check m3u8
	if keyframe {
		ok, err := w.m3u8.Update(packet.Timestamp)
		if err != nil {
			return err
		}
		// cut ts segment
		if ok {
			fn := w.m3u8.NextSegment()
			if err := w.tsMuxer.Reset(fn); err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Writer) packAudioPES(packet *avformat.AVPacket) error {
	audioTag, err := flv.ParseAACAudioData(packet.Body)
	if err != nil {
		return err
	}

	// cache audio extradata
	if packet.IsAACSeqHeader() {
		if err := w.aacParser.ParseAudioSpecificConfig(packet.Body); err != nil {
			return err
		}
	}

	// pes header
	dts := uint64(packet.Timestamp) * 90
	h := w.parseAudioPESHeader(audioTag, dts)
	if _, err := h.WriteTo(w.pesPacket); err != nil {
		return err
	}
	// pes payload
	if packet.IsAAC() {
		if _, err := w.aacParser.WriteADTS(uint16(packet.Length)); err != nil {
			return err
		}
		if _, err := w.aacParser.Write(audioTag.Data); err != nil {
			return err
		}
	}

	// TODO: propagate to ts muxer
	w.pesPacket.Reset()

	return nil
}

func (w *Writer) parseVideoPESHeader(vt *flv.VideoTagData, dts uint64) *PESHeader {
	pesHeader := &PESHeader{}
	pesHeader.PSCP = 0x000001
	pesHeader.SID = StreamIDVideo
	pesHeader.Flags1 = 0x80
	pesHeader.Flags2 = 0x80                             // maybe both dts & pts
	pesHeader.PHDL = 5                                  // len of dts & pts
	pesHeader.PTS = dts + uint64(vt.CompositionTime)*90 // cts = (pts - dts) / 90 (ms)
	pesHeader.DTS = dts

	// check should append dts
	if pesHeader.PTS != pesHeader.DTS {
		pesHeader.Flags2 |= 0x40
		pesHeader.PHDL = 10
	}
	// pes packet length
	if len(vt.Data)+int(pesHeader.PHDL)+3 > 0xFFFF {
		pesHeader.PPL = 0
	} else {
		pesHeader.PPL = uint16(len(vt.Data) + int(pesHeader.PHDL) + 3)
	}

	return pesHeader
}

func (w *Writer) parseAudioPESHeader(vt *flv.AudioTagData, dts uint64) *PESHeader {
	pesHeader := &PESHeader{}
	pesHeader.PSCP = 0x000001
	pesHeader.SID = StreamIDAudio
	if len(vt.Data)+5+3 > 0xFFFF {
		pesHeader.PPL = 0
	} else {
		pesHeader.PPL = uint16(len(vt.Data) + 5 + 3)
	}
	pesHeader.Flags1 = 0x80
	pesHeader.Flags2 = 0x80 // only dts
	pesHeader.PHDL = 5      // len of dts
	pesHeader.PTS = dts     // pts = dts
	pesHeader.DTS = dts
	return pesHeader
}
