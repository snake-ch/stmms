package avc

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
	NALUSTAPA                        = uint8(24) //
	NALUSTAPB                        = uint8(25) //
	NALUMTAP16                       = uint8(26) //
	NALUMTAP24                       = uint8(27) //
	NALUFUA                          = uint8(28) // FPU-A
	NALUFUB                          = uint8(29) // FPU-B
)

// slice type
const (
	SliceTypeP  = uint8(0)
	SliceTypeB  = uint8(1)
	SliceTypeI  = uint8(2)
	SliceTypeSP = uint8(3)
	SliceTypeSI = uint8(4)
)

var _StartCode = []byte{0x00, 0x00, 0x00, 0x01}
