package rtmp

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

// ChunkHeader .
//
// RTMP Chunk Header:
// +--------------+------------------+--------------------+------------+
// | Basic header | Chunk Msg Header | Extended TimeStamp | Chunk Data |
// +--------------+------------------+--------------------+------------+
//      1 byte      (0,3,7,11 bytes)      (0,4 bytes)
//
// Format = 0:
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | format | chunk stream id | timestamp delta | message length | msg type id | msg stream id |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// 	2 bits       6 bits           3 bytes          3 bytes         1 bytes         4 bytes
//
// Format = 1:
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | format | chunk stream id | timestamp delta | message length | msg type id |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// 	2 bits       6 bits            3 bytes         3 bytes         1 bytes
//
// Format = 2:
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-
// | format | chunk stream id | timestamp delta |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-
// 	2 bits       6 bits          3 bytes
//
// Format = 3:
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-
// | format | chunk stream id |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-
// 	2 bits       6 bits
//
type ChunkHeader struct {
	// Basic Header
	format        uint8
	chunkStreamID uint32

	// Chunk Message Header
	timestamp         uint32
	messageLength     uint32
	messageTypeID     uint8
	messageStreamID   uint32
	extendedTimestamp uint32

	// timestamp calculation
	calcTimestamp uint32
}

// GetTimestamp .
func (header *ChunkHeader) GetTimestamp() uint32 {
	if header.timestamp > 0xFFFFFF {
		return header.extendedTimestamp
	}
	return header.timestamp
}

// ReadFrom read chunk header from reader
func (header *ChunkHeader) ReadFrom(br *bufio.Reader) error {
	// base header
	baseHeader, err := br.ReadByte()
	if err != nil {
		return err
	}
	header.format = baseHeader >> 6
	header.chunkStreamID = uint32(baseHeader & 0x3f)

	// chunk stream id calculation
	switch header.chunkStreamID {
	case 0:
		// Chunk stream IDs 64-319 can be encoded in the 2-byte version
		// computed as (the second byte + 64)
		id2, err := br.ReadByte()
		if err != nil {
			return err
		}
		header.chunkStreamID = uint32(id2) + 64
	case 1:
		// Chunk stream IDs 64-65599 can be encoded in the 3-byte version
		// computed as ((the third byte)*256 + the second byte + 64).
		id2, err := br.ReadByte()
		if err != nil {
			return err
		}
		id3, err := br.ReadByte()
		if err != nil {
			return err
		}
		header.chunkStreamID = uint32(id3)*256 + uint32(id2) + 64
	default:
		// Chunk stream IDs 2-63 can be encoded in the 1-byte version
	}

	// chunk message header
	switch header.format {
	case 0: // chunk type-0, length equals 11 bytes
		buf := make([]byte, 11)
		if _, err := io.ReadFull(br, buf); err != nil {
			return err
		}
		header.timestamp = binary.BigEndian.Uint32(append([]byte{0x00}, buf[:3]...))
		header.messageLength = binary.BigEndian.Uint32(append([]byte{0x00}, buf[3:6]...))
		header.messageTypeID = buf[6]
		header.messageStreamID = binary.LittleEndian.Uint32(buf[7:])
	case 1: // chunk type-1, length equals 7 bytes
		buf := make([]byte, 7)
		if _, err := io.ReadFull(br, buf); err != nil {
			return err
		}
		header.timestamp = binary.BigEndian.Uint32(append([]byte{0x00}, buf[:3]...))
		header.messageLength = binary.BigEndian.Uint32(append([]byte{0x00}, buf[3:6]...))
		header.messageTypeID = buf[6]
	case 2: // chunk type-2, length equals 3 bytes
		buf := make([]byte, 3)
		if _, err := io.ReadFull(br, buf); err != nil {
			return err
		}
		header.timestamp = binary.BigEndian.Uint32(append([]byte{0x00}, buf[:3]...))
	case 3: // chunk type-3, length equals 0 bytes
	default:
		return fmt.Errorf("RTMP: chunk base header invalid format = %d", header.format)
	}

	// extended timestamp
	if header.timestamp == 0xFFFFFF {
		extendedTimestamp := make([]byte, 4)
		if _, err := io.ReadFull(br, extendedTimestamp); err != nil {
			return err
		}
		header.extendedTimestamp = binary.BigEndian.Uint32(extendedTimestamp)
	}
	return nil
}

// WriteTo write chunk header to writer
func (header *ChunkHeader) WriteTo(bw *bufio.Writer) error {
	// base header
	switch {
	case header.chunkStreamID < 64:
		// Chunk stream IDs 2-63 can be encoded in the 1-byte version
		err := bw.WriteByte(byte(header.format<<6 | byte(header.chunkStreamID)))
		if err != nil {
			return err
		}
	case header.chunkStreamID <= 319:
		// Chunk stream IDs 64-319 can be encoded in the 2-byte version
		// computed as (the second byte + 64)
		if err := bw.WriteByte(header.format << 6); err != nil {
			return err
		}
		if err := bw.WriteByte(byte(header.chunkStreamID - 64)); err != nil {
			return err
		}
	case header.chunkStreamID <= 65599:
		// Chunk stream IDs 64-65599 can be encoded in the 3-byte version
		// computed as ((the third byte)*256 + the second byte + 64).
		if err := bw.WriteByte(header.format<<6 | 0x01); err != nil {
			return err
		}
		if err := binary.Write(bw, binary.LittleEndian, uint16(header.chunkStreamID-64)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("RTMP: unsupport chunk stream ID large then 65599")
	}

	// chunk message header
	buf := make([]byte, 4)
	switch header.format {
	case 0: // type-0 = 11bytes, Timestamp + Message Length + Message Type + Message Stream ID
		binary.BigEndian.PutUint32(buf, header.timestamp)
		if _, err := bw.Write(buf[1:]); err != nil {
			return err
		}
		binary.BigEndian.PutUint32(buf, header.messageLength)
		if _, err := bw.Write(buf[1:]); err != nil {
			return err
		}
		if err := bw.WriteByte(header.messageTypeID); err != nil {
			return err
		}
		if err := binary.Write(bw, binary.LittleEndian, header.messageStreamID); err != nil {
			return err
		}
	case 1: // type-1 = 7 bytes, Timestamp + Message Length + Message Type
		binary.BigEndian.PutUint32(buf, header.timestamp)
		if _, err := bw.Write(buf[1:]); err != nil {
			return err
		}
		binary.BigEndian.PutUint32(buf, header.messageLength)
		if _, err := bw.Write(buf[1:]); err != nil {
			return err
		}
		if err := bw.WriteByte(header.messageTypeID); err != nil {
			return err
		}
	case 2: // type-2 = 3 bytes, Timestamp
		binary.BigEndian.PutUint32(buf, header.timestamp)
		if _, err := bw.Write(buf[1:]); err != nil {
			return err
		}
	case 3: // type-3 = 0 bytes
	}

	// extended timestamp
	if header.timestamp == 0xFFFFFF {
		err := binary.Write(bw, binary.BigEndian, header.extendedTimestamp)
		if err != nil {
			return err
		}
	}
	return nil
}
