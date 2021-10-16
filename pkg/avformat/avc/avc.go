package avc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// H.264/AVC
//
// Layer1 Format:
// 		Start Code + NAL Unit -> NALU Header + NALU Body
//		RTP Packet            -> NALU Header + NALU Body
//
// Layer2 NAL Unit:
//    NALU Body (RBSP) -> Slice -> Slice Header + Slice data
//
// Layer3 Slice:
// 		Slice data -> flags + Macroblock layer1 + Macroblock layer2 + ...
//
// Layer4 Slice data:
// 		Macroblock layer1 -> mb_type + PCM Data
// 		Macroblock layer2 -> mb_type + Sub_mb_pred or mb_pred + Residual Data
//
// Layer5:
// 		Residual Data -> Residual Block

// nal unit type, 24 - 31 Unspecified
const (
	NALUUnspecified                  = uint8(0)  // Unspecified
	NALUNonIDRPicture                = uint8(1)  // Coded slice of a non-IDR picture
	NALUDataPartitionA               = uint8(2)  // Coded slice data partition A
	NALUDataPartitionB               = uint8(3)  // Coded slice data partition B
	NALUDataPartitionC               = uint8(4)  // Coded slice data partition C
	NALUIDRPicture                   = uint8(5)  // Coded slice of an IDR picture
	NALUSEI                          = uint8(6)  // Supplemental enhancement information (SEI)
	NALUSPS                          = uint8(7)  // Sequence parameter set
	NALUPPS                          = uint8(8)  // Picture parameter set
	NALUAccessUnitDelimiter          = uint8(9)  // Access unit delimiter
	NALUSequenceEnd                  = uint8(10) // End of sequence
	NALUStreamEnd                    = uint8(11) // End of stream
	NALUFillerData                   = uint8(12) // Filler data
	NALUSPSExtension                 = uint8(13) // Sequence parameter set extension
	NALUPrefix                       = uint8(14) // Prefix NAL unit
	NALUSPSSubset                    = uint8(15) // Subset sequence parameter set
	NALUDPS                          = uint8(16) // Depth parameter set
	NALUReserved1                    = uint8(17) // Reserved
	NALUReserved2                    = uint8(18) // Reserved
	NALUNotAuxiliaryCoded            = uint8(19) // Coded slice of an auxiliary coded picture without partitioning
	NALUCodedSliceExtension          = uint8(20) // Coded slice extension
	NALUCodedSliceExtensionDepthView = uint8(21) // Coded slice extension for a depth view component or a 3D-AVC texture view component
	NALUReserved4                    = uint8(22) // Reserved
	NALUReserved5                    = uint8(23) // Reserved
)

// slice type
const (
	SliceTypeP  = uint8(0)
	SliceTypeB  = uint8(1)
	SliceTypeI  = uint8(2)
	SliceTypeSP = uint8(3)
	SliceTypeSI = uint8(4)
)

var StartCode = []byte{0x00, 0x00, 0x00, 0x01}
var StartCode3 = []byte{0x00, 0x00, 0x01}
var NaluAud = []byte{0x00, 0x00, 0x00, 0x01, 0x09, 0xf0}

// ----------------------------------------------------
// configurationVersion					[ 8b]
// AVCProfileIndication					[ 8b]
// profile_compatibility				[ 8b]
// AVCLevelIndication						[ 8b]
// reserved											[ 6b]
// LengthSizeMinusOne						[ 2b]
// reserved											[ 3b]
// numOfSequenceParameterSets 	[ 5b]
// -----loop----
// sequenceParameterSetLength		[16b]
// sequenceParameterSetNALUnits [n*8b]
// -----end-----
// numOfPictureParameterSets		[ 8b]
// -----loop----
// pictureParameterSetLength 		[16b]
// pictureParameterSetNALUnits  [n*8b]
// -----end-----
// ----------------------------------------------------

// AVCDecoderConfigurationRecord, see ISO_IEC_14496-15 5_2_4_1 setion
type AVCDecoderConfigurationRecord struct {
	ConfigurationVersion byte
	AvcProfileIndication byte
	ProfileCompatibility byte
	AvcLevelIndication   byte
	LengthSizeMinusOne   byte
	Sps                  []byte // sequenceParameterSetNALUnits
	Pps                  []byte // pictureParameterSetNALUnits
}

// TODO: consider sps & pps more than one
func (cfg *AVCDecoderConfigurationRecord) Bytes() []byte {
	bw := bytes.NewBuffer([]byte{})
	bw.WriteByte(cfg.ConfigurationVersion)
	bw.WriteByte(cfg.AvcProfileIndication)
	bw.WriteByte(cfg.ProfileCompatibility)
	bw.WriteByte(cfg.AvcLevelIndication)
	bw.WriteByte(0xFF)
	bw.WriteByte(0xE1)
	binary.Write(bw, binary.BigEndian, uint16(len(cfg.Sps)))
	bw.Write(cfg.Sps)
	bw.WriteByte(0x01)
	binary.Write(bw, binary.BigEndian, uint16(len(cfg.Pps)))
	bw.Write(cfg.Pps)
	return bw.Bytes()
}

// AVCParser .
type AVCParser struct {
	w         io.Writer
	extradata *AVCDecoderConfigurationRecord
	spspps    []byte
}

// NewAVCParser .
func NewAVCParser(w io.Writer) *AVCParser {
	return &AVCParser{
		w:         w,
		extradata: nil,
		spspps:    nil,
	}
}

// ParseExtradata parse AVCDecoderConfigurationRecord from avcC format
func (parser *AVCParser) ParseExtradata(p []byte) error {
	if len(p) < 5 {
		return fmt.Errorf("AVC: invalid length to parse extradata, len=%d", len(p))
	}

	extradata := &AVCDecoderConfigurationRecord{}
	extradata.ConfigurationVersion = p[0]
	extradata.AvcProfileIndication = p[1]
	extradata.ProfileCompatibility = p[2]
	extradata.AvcLevelIndication = p[3]
	extradata.LengthSizeMinusOne = p[4]&0x03 + 1 // NAL-Unit size

	// extract SPS
	var pos uint16 = 5
	numOfSps := int(p[pos] & 0x1F) // numOfSequenceParameterSets
	pos = pos + 1
	for idx := 0; idx < numOfSps; idx++ {
		lenOfSps := binary.BigEndian.Uint16(p[pos:]) // sequenceParameterSetLength
		pos = pos + 2
		extradata.Sps = append(extradata.Sps, p[pos:pos+lenOfSps]...)
		pos = pos + lenOfSps
	}

	// extract PPS
	numOfPps := int(p[pos] & 0x1F) // numOfPictureParameterSets
	pos = pos + 1
	for idx := 0; idx < numOfPps; idx++ {
		lenOfPps := binary.BigEndian.Uint16(p[pos:]) // pictureParameterSetLength
		pos = pos + 2
		extradata.Pps = append(extradata.Pps, p[pos:pos+lenOfPps]...)
		pos = pos + lenOfPps
	}

	parser.extradata = extradata
	return nil
}

// ----------------------------------------------------
//	avcC:
//	---------------
//	length	(UI32)
//	---------------
// 	nalu		UI8[N]
//	---------------
//	......
//	---------------
//
// nalu header:
//    0   1 2     3 4 5 6 7
//   +-+-+-+-+-+-+-+-+-+-+-+
//   | F |  NRI  |   Type  |
//   +-+-+-+-+-+-+-+-+-+-+-+
//   F: 	1bit, forbidden_zero_bit, must 0, discard if 1
//   NRI:	2bit, nal_ref_idc, I-frame/sps/pps = 3, P-frame = 2, B-frame = 0
//   Type:5bit, nal unit type
//
//	Annex-B:
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	| nalu(0x09) | 1byte | nalu |     | nalu(0x67) |     | nalu(0x68) |     | nalu(0x65) |      |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//       aud       0xf0                    SPS                PPS                 I
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	| nalu(0x09) | 1byte | nalu |     | nalu(0x41) |      |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//       aud       0xf0                     P

// WriteAnnexB write NALU from avcC format to annexB
func (parser *AVCParser) WriteAnnexB(avcC []byte) error {
	if len(avcC) < 4 {
		return fmt.Errorf("AVC: not enough byte to extract nal unit")
	}

	_, err := parser.w.Write(NaluAud)
	if err != nil {
		return err
	}

	hasWriteSpsPps := false
	for ridx := 0; ridx != len(avcC); {
		lenOfNalu := int(binary.BigEndian.Uint32(avcC[ridx:]))
		ridx += 4
		nalType := avcC[ridx] & 0x1F

		// cache SPS PPS, should not occur in avcC format ???
		switch nalType {
		case NALUSPS:
			fallthrough
		case NALUPPS:
			parser.spspps = append(StartCode, avcC[ridx:ridx+lenOfNalu]...)
		}

		// write Annex-B
		switch nalType {
		case NALUAccessUnitDelimiter:
		case NALUIDRPicture:
			if !hasWriteSpsPps {
				hasWriteSpsPps = true
				if err := parser.writeSpsPps(); err != nil {
					return err
				}
			}
			fallthrough
		case NALUNonIDRPicture:
			fallthrough
		case NALUSEI:
			// TODO: consider N-slice using strat_code_3bytes
			if _, err := parser.w.Write(StartCode); err != nil {
				return err
			}
			if _, err := parser.w.Write(avcC[ridx : ridx+lenOfNalu]); err != nil {
				return err
			}
		}
		ridx += lenOfNalu
	}
	return nil
}

func (parser *AVCParser) writeSpsPps() (err error) {
	if parser.spspps != nil {
		if _, err := parser.w.Write(parser.spspps); err != nil {
			return err
		}
		return nil
	}

	if parser.extradata != nil {
		// sps nalu
		if _, err := parser.w.Write(StartCode); err != nil {
			return err
		}
		if _, err := parser.w.Write(parser.extradata.Sps); err != nil {
			return err
		}
		// pps nalu
		if _, err := parser.w.Write(StartCode); err != nil {
			return err
		}
		if _, err := parser.w.Write(parser.extradata.Pps); err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("AVC Parser: no extradata or cache SPS/PPS")
}
