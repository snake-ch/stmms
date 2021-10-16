package rtp

const (
	PacketTypeAVC = 96
	PacketTypeAAC = 97
)

// RFC3984 Table 1.
//
// Type   Packet    Type name                        Section
// ---------------------------------------------------------
// 0      undefined                                    -
// 1-23   NAL unit  Single NAL unit packet per H.264   5.6
// 24     STAP-A    Single-time aggregation packet     5.7.1
// 25     STAP-B    Single-time aggregation packet     5.7.1
// 26     MTAP16    Multi-time aggregation packet      5.7.2
// 27     MTAP24    Multi-time aggregation packet      5.7.2
// 28     FU-A      Fragmentation unit                 5.8
// 29     FU-B      Fragmentation unit                 5.8
// 30-31  undefined                                    -

const (
	NALUSTAPA  = uint8(24)
	NALUSTAPB  = uint8(25)
	NALUMTAP16 = uint8(26)
	NALUMTAP24 = uint8(27)
	NALUFUA    = uint8(28)
	NALUFUB    = uint8(29)
)
