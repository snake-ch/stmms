package rtcp

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	SR   = uint8(200)
	RR   = uint8(201)
	SDES = uint8(202)
	BYE  = uint8(203)
	APP  = uint8(204)
)
const ReportBlockSize = 32 * 6
const FixedNTP = (70*365 + 17) * 24 * 60 * 60

type Packet struct {
	*Header
	Payload []byte
}

func ParsePacket(p []byte) (*Packet, error) {
	packet := &Packet{
		Header:  &Header{},
		Payload: nil,
	}
	n, err := packet.Header.Parse(p)
	if err != nil {
		return nil, err
	}
	if packet.Header.Length >= 1 {
		packet.Payload = p[n:]
	}
	return packet, nil
}

// RTCP header
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |V=2|P|  RC   |      PT=SR    |            length             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

type Header struct {
	V      uint8  // [ 2b] version
	P      uint8  // [ 1b] padding
	RC     uint8  // [ 5b] reception report count
	PT     uint8  // [ 8b] packet type
	Length uint16 // [16b] length
	SSRC   uint32 // [32b] sender ssrc
}

func (header *Header) Parse(p []byte) (n int, err error) {
	if len(p) < 4 {
		return 0, fmt.Errorf("RTCP: not enough bytes to parse, length: %d", len(p))
	}
	header.V = p[0] >> 6
	header.P = (p[0] >> 5) & 0x01
	header.RC = p[0] & 0x1F
	header.PT = p[1]
	header.Length = binary.BigEndian.Uint16(p[2:])
	return 4, nil
}

// SR sender report
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |V=2|P|  RC   |    PT=SR=200  |            length             |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                       SSRC of sender                        |
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |             NTP timestamp, most significant word            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |            NTP timestamp, least significant word            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                        RTP timestamp                        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                    sender’s packet count                    |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                    sender’s octet count                     |
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |                SSRC_1 (SSRC of first source)                |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |fraction lost|       cumulative number of packets lost       |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |          extended highest sequence number received          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                     interarrival jitter                     |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                        last SR (LSR)                        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 delay since last SR (DLSR)                  |
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |               SSRC_2 (SSRC of second source)                |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// :                             ...                             :
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |                 profile-specific extensions                 |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

type SenderReport struct {
	*Header
	MSW         uint32         // [32b] most significant word
	LSW         uint32         // [32b] least significant word
	TS          uint32         // [32b]	rtp timestamp
	PacketCount uint32         // [32b]
	OctetCount  uint32         // [32b]
	Blocks      []*ReportBlock // [6*32b]
}

type ReportBlock struct {
	SSRC     uint32 // [32b]
	Fraction uint8  // [ 8b]
	Lost     uint32 // [24b]
	SN       uint32 // [32b]
	Jitter   uint32 // [32b]
	LSR      uint32 // [32b]
	DLSR     uint32 // [32b]
}

// parse sender report
func (packet *Packet) ParseSR() *SenderReport {
	p := packet.Payload

	sr := &SenderReport{
		Header: packet.Header,
		Blocks: make([]*ReportBlock, 0),
	}
	sr.SSRC = binary.BigEndian.Uint32(p[0:])
	sr.MSW = binary.BigEndian.Uint32(p[4:])
	sr.LSW = binary.BigEndian.Uint32(p[8:])
	sr.TS = binary.BigEndian.Uint32(p[12:])
	sr.PacketCount = binary.BigEndian.Uint32(p[16:])
	sr.OctetCount = binary.BigEndian.Uint32(p[20:])
	// blocks
	if sr.Header.Length > 6 {
		pos := 24
		numOfBlock := (len(p) - pos) / ReportBlockSize
		for idx := 0; idx < numOfBlock; idx++ {
			block := &ReportBlock{}
			block.SSRC = binary.BigEndian.Uint32(p[pos:])
			block.Fraction = p[pos+4]
			block.Lost = uint32(p[pos+5])<<16 + uint32(p[pos+6])<<8 + uint32(p[pos+7])
			block.SN = binary.BigEndian.Uint32(p[pos+8:])
			block.Jitter = binary.BigEndian.Uint32(p[pos+12:])
			block.LSR = binary.BigEndian.Uint32(p[pos+16:])
			block.DLSR = binary.BigEndian.Uint32(p[pos+20:])
			sr.Blocks = append(sr.Blocks, block)
			pos += 24
		}
	}
	return sr
}

// msw and lsw as ntp timestamp, return in nano
func (sr *SenderReport) NTP() uint64 {
	return uint64(sr.MSW-FixedNTP)*1e9 + uint64(sr.LSW)*1e9>>32
}

// RR receiver report
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |V=2|P|  RC   |    PT=SR=201  |            length             |
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |                SSRC_1 (SSRC of first source)                |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |fraction lost|       cumulative number of packets lost       |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |          extended highest sequence number received          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                     interarrival jitter                     |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                        last SR (LSR)                        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 delay since last SR (DLSR)                  |
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |               SSRC_2 (SSRC of second source)                |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// :                             ...                             :
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |                 profile-specific extensions                 |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

type ReceiverReport struct {
	*Header
	blocks []*ReportBlock
}

func (rr *ReceiverReport) WriteTo(w io.Writer) (int64, error) {
	n := 0
	return int64(n), nil
}
