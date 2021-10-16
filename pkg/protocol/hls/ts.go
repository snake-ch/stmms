package hls

import (
	"encoding/binary"
	"fmt"
)

// TSHeader
// ------------------------------------------------
// sync_byte                    [8b]
// transport_error_indicator    [1b]
// payload_unit_start_indicator [1b]
// transport_priority           [1b]
// PID                          [13b]
// transport_scrambling_control [2b]
// adaptation_field_control     [2b]
// continuity_counter           [4b]
// ------------------------------------------------
type TSHeader struct {
	SB   uint8  // [8b ] sync_byte
	TEI  uint8  // [1b ] transport_error_indicator
	PUSI uint8  // [1b ] payload_unit_start_indicator
	TP   uint8  // [1b ] transport_priority
	PID  uint16 // [13b] PID
	TSC  uint8  // [2b ] transport_scrambling_control
	AFC  uint8  // [2b ] adaptation_field_control
	CC   uint8  // [4b ] continuity_counter
}

// Parse
func (header *TSHeader) Parse(p []byte) error {
	if len(p) < 4 {
		return fmt.Errorf("TS: invalid length to parse header")
	}

	header.SB = p[0]
	if header.SB != 0x47 {
		return fmt.Errorf("TS: SYNC error, should be 0x47")
	}
	header.TEI = p[1] >> 7
	header.PUSI = p[1] >> 6 & 0x01
	header.TP = p[1] >> 5 & 0x01
	header.PID = uint16(p[1]&0x1F)<<8 | uint16(p[2])
	header.TSC = p[3] >> 6
	header.AFC = p[3] >> 4 & 0x03
	header.CC = p[3] & 0x0F
	return nil
}

// Write
func (tsh *TSHeader) Write(buf []byte) error {
	if len(buf) < 4 {
		return fmt.Errorf("TS: not enough length to write")
	}
	buf[0] = 0x47
	buf[1] = (tsh.TEI & 0x01) << 7
	buf[1] |= (tsh.PUSI & 0x01) << 6
	buf[1] |= (tsh.TP & 0x01) << 5
	buf[1] |= uint8((tsh.PID >> 8) & 0x1F)
	buf[2] = uint8(tsh.PID & 0xFF)
	buf[3] = (tsh.TSC & 0x03) << 6
	buf[3] |= (tsh.AFC & 0x03) << 4
	buf[3] |= tsh.CC & 0x0F
	return nil
}

// TSAdaptation
// ----------------------------------------------------------
// adaptation_field_length              [8b]
// discontinuity_indicator              [1b]
// random_access_indicator              [1b]
// elementary_stream_priority_indicator [1b]
// PCR_flag                             [1b]
// OPCR_flag                            [1b]
// splicing_point_flag                  [1b]
// transport_private_data_flag          [1b]
// adaptation_field_extension_flag      [1b]
// -----if PCR_flag == 1-----
// program_clock_reference_base         [33b]
// reserved                             [6b]
// program_clock_reference_extension    [9b]
// ----------------------------------------------------------
type TSAdaptation struct {
	Length uint8  // [8b ] adaptation_field_length
	Flags  uint8  // [8b ]
	PCR    uint16 // [48b] program_clock_reference
}

// PAT Program Association Table
// ----------------------------------------------------------
// table_id                 [8b]
// section_syntax_indicator [1b]
// '0'                      [1b]
// reserved                 [2b]
// section_length           [12b]
// transport_stream_id      [16b]
// reserved                 [2b]
// version_number           [5b]
// current_next_indicator   [1b]
// section_number           [8b]
// last_section_number      [8b]
// -----loop-----
// program_number           [16b]
// reserved                 [3b]
// program_map_PID          [13b]
// -----end------
// CRC_32                   [32b]
// ----------------------------------------------------------
type PAT struct {
	TID      uint8         // [8b ] table_id
	SSI      uint8         // [1b ] section_syntax_indicator
	SL       uint16        // [12b] section_length
	TSI      uint16        // [16b] transport_stream_id
	VN       uint8         // [5b ] version_number
	CNI      uint8         // [1b ] current_next_indicator
	SN       uint8         // [8b ] section_number
	LSN      uint8         // [8b ] last_section_number
	Programs []*PATProgram // [32b] *N
	CRC      uint32        // [32b] CRC_32
}

type PATProgram struct {
	PN  uint16 // [16b] program_number
	PID uint16 // [13b] network_PID or program_map_PID
}

// Parse
func (pat *PAT) Parse(p []byte) error {
	if len(p) < 12 {
		return fmt.Errorf("TS PAT: not enough bytes to parse")
	}

	pat.TID = p[0]
	pat.SSI = p[1] >> 7
	pat.SL = uint16(p[1]&0x0F)<<8 | uint16(p[2])
	pat.TSI = uint16(p[3])<<8 | uint16(p[4])
	pat.VN = p[5] >> 1 & 0x1F
	pat.CNI = p[5] & 0x01
	pat.SN = p[6]
	pat.LSN = p[7]
	pat.CRC = binary.BigEndian.Uint32(p[pat.SL+3-4:])

	pat.Programs = make([]*PATProgram, 0)
	for idx := uint16(0); idx < pat.SL-9; idx += 4 {
		program := &PATProgram{}
		program.PN = uint16(p[8+idx])<<8 | uint16(p[9+idx])
		program.PID = uint16(p[10+idx]&0x1F)<<8 | uint16(p[11+idx])
		if program.PN == 0 {
			// NIT
		} else {
			pat.Programs = append(pat.Programs, program) // PMT
		}
	}
	return nil
}

// PMT Program Map Table
// ----------------------------------------
// table_id                 [8b]
// section_syntax_indicator [1b]
// 0                        [1b]
// reserved                 [2b]
// section_length           [12b]
// program_number           [16b]
// reserved                 [2b]
// version_number           [5b]
// current_next_indicator   [1b]
// section_number           [8b]
// last_section_number      [8b]
// reserved                 [3b]
// PCR_PID                  [13b]
// reserved                 [4b]
// program_info_length      [12b]
// -----loop-----
// stream_type              [8b]
// reserved                 [3b]
// elementary_PID           [13b]
// reserved                 [4b]
// ES_info_length_length    [12b]
// --------------
// CRC32                    [32b]
// ----------------------------------------
type PMT struct {
	TID        uint8        // [8b ] table_id
	SSI        uint8        // [1b ] section_syntax_indicator
	SL         uint16       // [12b] section_length
	PN         uint16       // [16b] program_number
	VN         uint8        // [5b ] version_number
	CNI        uint8        // [1b ] current_next_indicator
	SN         uint8        // [8b ] section_number
	LSN        uint8        // [8b ] last_section_number
	PCR_PID    uint16       // [13b] PCR_PID
	PIL        uint16       // [12b] program_info_length
	PMTStreams []*PMTStream // [32b] *N
	CRC        uint32       // [32b] CRC_32
}

type PMTStream struct {
	ST  uint8  // [8b ] stream_type
	PID uint16 // [13b] elementary_PID
	EIL uint16 // [12b] ES_info_length
}

func (pmt *PMT) Parse(p []byte) error {
	if len(p) < 16 {
		return fmt.Errorf("TS PAT: not enough bytes to parse")
	}

	pmt.TID = p[0]
	pmt.SSI = p[1] >> 7
	pmt.SL = uint16((p[1]&0x0F))<<8 | uint16(p[2])
	pmt.PN = uint16(p[3])<<8 | uint16(p[4])
	pmt.VN = p[5] >> 1 & 0x1F
	pmt.CNI = p[5] & 0x01
	pmt.SN = p[6]
	pmt.LSN = p[7]
	pmt.PCR_PID = uint16(p[8]&0x1F)<<8 | uint16(p[9])
	pmt.PIL = uint16(p[10]&0x0F)<<8 | uint16(p[11])
	pmt.CRC = binary.BigEndian.Uint32(p[pmt.SL+3-4:])

	// skip program info
	pos := uint16(12)
	if pmt.PIL != 0 {
		pos += pmt.PIL
	}

	// streams
	for idx := uint16(0); pos < pmt.SL-13; idx += 5 {
		stream := &PMTStream{}
		stream.ST = p[pos+idx]
		stream.PID = uint16(p[pos+idx+1]&0x1F)<<8 | uint16(p[pos+idx+2])
		stream.EIL = uint16(p[pos+idx+3]&0x0F)<<8 | uint16(p[pos+idx+4])
		// skip stream info
		if stream.EIL != 0 {
			idx += stream.EIL
		}
		pmt.PMTStreams = append(pmt.PMTStreams, stream)
	}
	return nil
}
