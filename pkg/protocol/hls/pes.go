package hls

import (
	"fmt"
	"io"
)

// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | pes header |   optional pes header  |      pes payload      |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// 		6Byte             3~259Byte              max 65526Byte

// PES Packet Elementary Stream
// -----------------------------------------------------------
// packet_start_code_prefix  [24b]	0x000001
// stream_id                 [8b]
// PES_packet_length         [16b]
// '10'                      [2b]
// PES_scrambling_control    [2b]
// PES_priority              [1b]
// data_alignment_indicator  [1b]
// copyright                 [1b]
// original_or_copy          [1b]
// PTS_DTS_flags             [2b]
// ESCR_flag                 [1b]
// ES_rate_flag              [1b]
// DSM_trick_mode_flag       [1b]
// additional_copy_info_flag [1b]
// PES_CRC_flag              [1b]
// PES_extension_flag        [1b]
// PES_header_data_length    [8b]
// -----------------------------------------------------------
type PESHeader struct {
	PSCP   uint32 // [24b] packet_start_code_prefix
	SID    uint8  // [8b ] stream_id
	PPL    uint16 // [16b] PES_packet_length
	Flags1 uint8  // [8b ]
	Flags2 uint8  // [8b ]
	PHDL   uint8  // [8b ] PES_header_data_length
	PTS    uint64 // [33b]
	DTS    uint64 // [33b]
}

func (pes *PESHeader) Parse(p []byte) error {
	if len(p) < 9 {
		return fmt.Errorf("TS PAT: not enough bytes to parse")
	}
	pes.PSCP = uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
	pes.SID = p[3]
	pes.PPL = uint16(p[4])<<8 | uint16(p[5])
	pes.Flags1 = p[6]
	pes.Flags2 = p[7]
	pes.PHDL = p[8]
	// pts
	if pes.Flags2>>7 == 0x01 {
		pes.PTS = uint64(p[9]>>1&0x07) << 30
		pes.PTS |= (uint64(p[10])<<8 | uint64(p[11])) >> 1 << 15
		pes.PTS |= (uint64(p[12])<<8 | uint64(p[13])) >> 1
	}
	// dts
	if (pes.Flags2<<1)>>7 == 0x01 {
		pes.DTS = uint64(p[12]>>1&0x07) << 30
		pes.DTS |= (uint64(p[13])<<8 | uint64(p[14])) >> 1 << 15
		pes.DTS |= (uint64(p[15])<<8 | uint64(p[16])) >> 1
	}
	return nil
}

func (pes *PESHeader) WriteTo(w io.Writer) (int64, error) {
	if pes.PSCP != 0x000001 {
		return 0, fmt.Errorf("PES: start prefix code error, must be 0x000001")
	}
	buf := make([]byte, 19)
	buf[0] = 0x00
	buf[1] = 0x00
	buf[2] = 0x01
	buf[3] = pes.SID
	buf[4] = uint8(pes.PPL >> 8)
	buf[5] = uint8(pes.PPL & 0xFF)
	buf[6] = pes.Flags1
	buf[7] = pes.Flags2
	buf[8] = pes.PHDL
	writePTSDTS(buf[9:], pes.Flags2>>6, pes.PTS)

	if pes.Flags2>>6 == 0x03 {
		writePTSDTS(buf[9+5:], 1, pes.DTS)
		n, err := w.Write(buf)
		return int64(n), err
	}

	n, err := w.Write(buf[:9+5])
	return int64(n), err
}

func writePTSDTS(p []byte, flag uint8, ts uint64) {
	p[0] = (flag << 4) | (uint8(ts>>30) & 0x07) | 1

	t := (((ts >> 15) & 0x7FFF) << 1) | 1
	p[1] = uint8(t >> 8)
	p[2] = uint8(t)

	t = ((ts & 0x7FFF) << 1) | 1
	p[3] = uint8(t >> 8)
	p[4] = uint8(t)
}
