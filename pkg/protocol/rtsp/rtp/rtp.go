package rtp

import (
	"encoding/binary"
	"fmt"
)

const RTPHeaderLength = 12

// RTP Fixed Header
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |V=2|P|X|  CC   |M|     PT      |       sequence number         |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                           timestamp                           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |           synchronization source (SSRC) identifier            |
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |            contributing source (CSRC) identifiers             |
// |                             ....                              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |       defined by profile      |             length            |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                       header extension                        |
// |                             ....                              |

type Header struct {
	v    uint8    // [ 2b]		version
	p    uint8    // [ 1b]		padding
	x    uint8    // [ 1b]		extension
	cc   uint8    // [ 4b]		csrc_count
	m    uint8    // [ 1b]		marker
	pt   uint8    // [ 7b]		packet_type
	sn   uint16   // [16b]		sequence number
	ts   uint32   // [32b]		timestamp
	ssrc uint32   // [32b]		synchronization source
	csrc []uint32 // [n*32b]	contributing source
	extension
}

type extension struct {
	profile uint16
	length  uint16
	data    []byte
}

func (header *Header) Parse(p []byte) (n int, err error) {
	if len(p) < RTPHeaderLength {
		return 0, fmt.Errorf("RTP: invalid length to parse header, len=%d", len(p))
	}
	header.v = p[0] >> 6
	header.p = (p[0] >> 5) & 0x01
	header.x = (p[0] >> 4) & 0x01
	header.cc = p[0] & 0x0F
	header.m = p[1] >> 7
	header.pt = p[1] & 0x7F
	header.sn = binary.BigEndian.Uint16(p[2:])
	header.ts = binary.BigEndian.Uint32(p[4:])
	header.ssrc = binary.BigEndian.Uint32(p[8:])

	// csrc
	pos := RTPHeaderLength
	header.csrc = make([]uint32, header.cc)
	for idx := range header.csrc {
		header.csrc[idx] = binary.BigEndian.Uint32(p[pos:])
		pos += 4
	}

	// header extension
	if header.x == 0x01 {
		header.profile = binary.BigEndian.Uint16(p[pos:])
		pos += 2
		header.length = binary.BigEndian.Uint16(p[pos:])
		pos += 2
		if header.length > 0 {
			header.data = p[pos : pos+4*int(header.length)]
			pos += 4 * int(header.length)
		}
	}

	return pos, nil
}

type Packet struct {
	header  *Header
	payload []byte
}

func ParsePacket(p []byte) (*Packet, error) {
	packet := &Packet{
		header:  &Header{},
		payload: nil,
	}

	n, err := packet.header.Parse(p)
	if err != nil {
		return nil, err
	}
	packet.payload = p[n:]

	return packet, nil
}

func (packet *Packet) Timestamp() uint32 {
	return packet.header.ts
}
