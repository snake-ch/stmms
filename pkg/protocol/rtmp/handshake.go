package rtmp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"time"
)

const (
	_Timeout                           = 5 * time.Second
	_SizeC0, _SizeS0                   = 1, 1
	_SizeC1, _SizeC2, _SizeS1, _SizeS2 = 1536, 1536, 1536, 1536
	_SizeDigest                        = 32
)

var (
	_FmsVersion = []byte{0x04, 0x05, 0x00, 0x00}
	_FpVersion  = []byte{0x09, 0x00, 0x7C, 0x02} // ffmpeg version

	_GenuineFpKey = []byte{
		'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
		'F', 'l', 'a', 's', 'h', ' ', 'P', 'l', 'a', 'y', 'e', 'r', ' ',
		'0', '0', '1', // Genuine Adobe Flash Player 001
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
		0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
	_GenuineFmsKey = []byte{
		'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
		'F', 'l', 'a', 's', 'h', ' ', 'M', 'e', 'd', 'i', 'a', ' ',
		'S', 'e', 'r', 'v', 'e', 'r', ' ',
		'0', '0', '1', // Genuine Adobe Flash Media Server 001
		0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
		0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
		0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
	}
)

// SimpleClientHandshake .
//
//  |   client   |   Server   |
//  | ------- C0 + C1 ------> |
//  | <---- S0 + S1 +S2 ----- |
//  | --------- C2 ---------> |
//
// C1/S1: time(4bytes) + zero(4bytes) + random-data(1528bytes)
//
func (nc *NetConnection) SimpleClientHandshake() error {
	// Send C0
	nc.rw.WriteByte(0x03)
	// Send C1
	c1, _ := generateRandom(_SizeC1)
	binary.BigEndian.PutUint32(c1, uint32(time.Now().UnixNano()/1e6)) // timestamp
	copy(c1[5:], []byte{0x00, 0x00, 0x00, 0x00})                      // zero
	nc.rw.Write(c1)
	nc.rw.Flush()

	// Read S0
	if s0, err := nc.rw.ReadByte(); s0 != 0x03 {
		return err
	}
	// Read S1
	s1 := make([]byte, _SizeS1)
	if _, err := io.ReadFull(nc.rw, s1); err != nil {
		return err
	}
	// Read S2
	s2 := make([]byte, _SizeS2)
	if _, err := io.ReadFull(nc.rw, s2); err != nil {
		return err
	}

	return nil
}

// ComplexClientHandshake .
//
//  |   client   |   Server   |
//  | ------- C0 + C1 ------> |
//  | <---- S0 + S1 +S2 ----- |
//  | --------- C2 ---------> |
//
// C1/S1:
// 		schemal-0 = time(4bytes) + version(4bytes) + key(764bytes) + digest(764bytes)
// 		schemal-1 = time(4bytes) + version(4bytes) + digest(764bytes) + key(764bytes)
//
// C2/S2
// 		time(4bytes) + time2(4bytes) + random-data(1496bytes) + + digest(32bytes)
//
func (nc *NetConnection) ComplexClientHandshake() error {
	// Send C0
	if err := nc.rw.WriteByte(0x03); err != nil {
		return fmt.Errorf("%0s c0 error, %s", "S -> C", err)
	}

	// Send C1
	c1, _ := generateRandom(_SizeC1)
	binary.BigEndian.PutUint32(c1, uint32(time.Now().UnixNano()/1e6)) // timestamp
	copy(c1[5:], _FpVersion)                                          // version
	c1DigestPos := calcDigestPosition(c1, 8)                          // schemal-1
	c1Digest, _ := generateDigest(c1, 8, _GenuineFpKey[:30])          // digest
	copy(c1[c1DigestPos:], c1Digest)
	nc.goConn.SetWriteDeadline(time.Now().Add(_Timeout))
	if _, err := nc.rw.Write(c1); err != nil {
		return err
	}
	if err := nc.rw.Flush(); err != nil {
		return fmt.Errorf("%0s c1 error, %s", "S -> C", err)
	}

	// Read S0
	nc.goConn.SetReadDeadline(time.Now().Add(_Timeout))
	s0, err := nc.rw.ReadByte()
	if err != nil {
		return nil
	}
	if s0 != 0x03 {
		return fmt.Errorf("%0s s0 error, got s0: %x", "C -> S", s0)
	}

	// Read S1
	s1 := make([]byte, _SizeS1)
	nc.goConn.SetReadDeadline(time.Now().Add(_Timeout))
	if _, err := io.ReadFull(nc.rw, s1); err != nil {
		return err
	}
	s1DigestPos := findDigest(s1, 8, _GenuineFmsKey[0:36]) // check scheme-0
	if s1DigestPos == -1 {
		s1DigestPos = findDigest(s1, 8, _GenuineFmsKey[0:36]) // check scheme-1
		if s1DigestPos == -1 {
			return fmt.Errorf("%0s s1 scheme validating failed", "C -> S")
		}
	}
	s1ArrivedTime := uint32(time.Now().UnixNano() / 1e6) // S1 arrived time, using in C2

	// Read S2
	nc.goConn.SetReadDeadline(time.Now().Add(_Timeout))
	s2 := make([]byte, _SizeS2)
	if _, err := io.ReadFull(nc.rw, s2); err != nil {
		return err
	}
	secret, _ := hmacSha256(c1[c1DigestPos:c1DigestPos+_SizeDigest], _GenuineFmsKey)
	s2Digest, _ := hmacSha256(s2[:_SizeS2-_SizeDigest], secret)
	if bytes.Compare(s2Digest, s2[_SizeS2-_SizeDigest:]) != 0 {
		return fmt.Errorf("%0s s2 digest mismatch", "C -> S")
	}

	// Send C2
	c2, _ := generateRandom(_SizeC2)
	secret, _ = hmacSha256(s1[s1DigestPos:s1DigestPos+_SizeDigest], _GenuineFpKey)
	c2Digest, _ := hmacSha256(c2[:_SizeC2-_SizeDigest], secret)
	copy(c2, s1[0:4])                             // time1
	binary.BigEndian.PutUint32(c2, s1ArrivedTime) // time2
	copy(c2[_SizeC2-_SizeDigest:], c2Digest)      // digest
	nc.goConn.SetWriteDeadline(time.Now().Add(_Timeout))
	if _, err := nc.rw.Write(c2); err != nil {
		return err
	}
	if err := nc.rw.Flush(); err != nil {
		return fmt.Errorf("%0s c2 error", "S -> C")
	}

	nc.goConn.SetDeadline(time.Time{})
	return nil
}

// ServerHandshake .
//
//  |   client   |   Server   |
//  | ------- C0 + C1 ------> |
//  | <---- S0 + S1 +S2 ----- |
//  | --------- C2 ---------> |
//
// C1/S1:
// 		schemal-0 = time(4bytes) + version(4bytes) + key(764bytes) + digest(764bytes)
// 		schemal-1 = time(4bytes) + version(4bytes) + digest(764bytes) + key(764bytes)
//
// C2/S2
// 		time(4bytes) + time2(4bytes) + random(1496bytes) + + digest(32bytes)
//
func (nc *NetConnection) ServerHandshake() error {
	// Read C0
	nc.goConn.SetReadDeadline(time.Now().Add(_Timeout))
	c0, err := nc.rw.ReadByte()
	if err != nil {
		return err
	}
	if c0 != 0x03 {
		return fmt.Errorf("%0s c0 error, got c0: %x", "C -> S", c0)
	}

	// Read C1
	c1 := make([]byte, _SizeC1)
	nc.goConn.SetReadDeadline(time.Now().Add(_Timeout))
	if _, err := io.ReadFull(nc.rw, c1); err != nil {
		return err
	}
	c1ArrivedTime := uint32(time.Now().UnixNano() / 1e6) // C1 arrived time, using in S2

	/***********************************
	 ********* simple handshake ********
	 ***********************************/
	c1Version := binary.BigEndian.Uint32(c1[4:8])
	if c1Version == 0 {
		// make S1 adn S2 equals C1
		s1, s2 := make([]byte, _SizeS1), make([]byte, _SizeS2)
		copy(s1, c1)
		copy(s2, c1)

		// Send S0 + S1 + S2
		nc.goConn.SetWriteDeadline(time.Now().Add(_Timeout))
		if err := nc.rw.WriteByte(0x03); err != nil {
			return err
		}
		if _, err := nc.rw.Write(s1); err != nil {
			return err
		}
		if _, err := nc.rw.Write(s2); err != nil {
			return err
		}
		if err := nc.rw.Flush(); err != nil {
			return err
		}

		// Read c2
		nc.goConn.SetReadDeadline(time.Now().Add(_Timeout))
		c2 := make([]byte, _SizeC2)
		if _, err := io.ReadFull(nc.rw, c2); err != nil {
			return err
		}

		nc.goConn.SetDeadline(time.Time{})
		return nil
	}

	/***********************************
	 ******** complex handshake ********
	 ***********************************/
	c1DigestPos := findDigest(c1, 8, _GenuineFpKey[0:30]) // check if scheme-0
	if c1DigestPos == -1 {
		c1DigestPos = findDigest(c1, 764+8, _GenuineFpKey[0:30]) // check if scheme-1
		if c1DigestPos == -1 {
			return fmt.Errorf("%0s c1 scheme validating failed", "C -> S")
		}
	}

	// Send S0
	if err := nc.rw.WriteByte(0x03); err != nil {
		return fmt.Errorf("%0s s0 error", "S -> C")
	}

	// Send S1
	s1, _ := generateRandom(_SizeS1)
	binary.BigEndian.PutUint32(s1, uint32(time.Now().UnixNano()/1e6)) // timestamp
	copy(s1[4:], _FmsVersion)                                         // version
	s1DigestPos := calcDigestPosition(s1, 8)                          // schemal-1
	s1Digest, _ := generateDigest(s1, 8, _GenuineFmsKey[:36])         // digest
	copy(s1[s1DigestPos:], s1Digest)
	nc.goConn.SetWriteDeadline(time.Now().Add(_Timeout))
	if _, err := nc.rw.Write(s1); err != nil {
		return fmt.Errorf("%0s s1 error", "S -> C")
	}

	// Send S2
	s2, _ := generateRandom(_SizeS2)
	secret, _ := hmacSha256(c1[c1DigestPos:c1DigestPos+_SizeDigest], _GenuineFmsKey)
	copy(s2, c1[0:4])                                           // time
	binary.BigEndian.PutUint32(s2[4:8], c1ArrivedTime)          // time2
	s2Digest, _ := hmacSha256(s2[:_SizeS2-_SizeDigest], secret) // digest
	copy(s2[_SizeS2-_SizeDigest:], s2Digest)                    // digest
	nc.goConn.SetWriteDeadline(time.Now().Add(_Timeout))
	if _, err := nc.rw.Write(s2); err != nil {
		return err
	}
	if err := nc.rw.Flush(); err != nil {
		return fmt.Errorf("%0s s2 error", "S -> C")
	}

	// Read C2
	c2 := make([]byte, _SizeC2)
	nc.goConn.SetReadDeadline(time.Now().Add(_Timeout))
	if _, err := io.ReadFull(nc.rw, c2); err != nil {
		return fmt.Errorf("%0s c2 error", "C -> S")
	}
	// TODO: completed C2 validation
	if false {
		secret, _ = hmacSha256(s1[s1DigestPos:s1DigestPos+_SizeDigest], _GenuineFpKey)
		c2Digest, _ := hmacSha256(c2[:_SizeC2-_SizeDigest], secret)
		if bytes.Compare(c2Digest, c2[_SizeS2-_SizeDigest:]) != 0 {
			return fmt.Errorf("%0s c2 digest mismatch", "C -> S")
		}
	}

	nc.goConn.SetDeadline(time.Time{})
	return nil
}

// make specified size random byte array
func generateRandom(size int) ([]byte, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return nil, err
	}
	return data, nil
}

// generate data digest using sha256
func hmacSha256(data []byte, secret []byte) ([]byte, error) {
	hash := hmac.New(sha256.New, secret)
	_, err := hash.Write(data)
	if err != nil {
		return nil, err
	}
	return hash.Sum(nil), nil
}

// get digest offset position
func calcDigestPosition(data []byte, offset int) int {
	position := int(data[offset])
	position = position + int(data[offset+1])
	position = position + int(data[offset+2])
	position = position + int(data[offset+3])
	return (position % 728) + offset + 4
}

// compare digest and return position if matched, orelse return -1
func findDigest(data []byte, offset int, secret []byte) int {
	digestPos := calcDigestPosition(data, offset)
	hash, _ := generateDigest(data, offset, secret)
	if bytes.Compare(hash, data[digestPos:digestPos+_SizeDigest]) == 0 {
		return digestPos
	}
	return -1
}

// DIGEST(764bytes):
//		offset        -> 4bytes
//		random-data-1 -> (offset)bytes
//		digest-data   -> 32bytes
//		random-data-2 -> (764-4-offset-32)bytes
func generateDigest(data []byte, offset int, secret []byte) ([]byte, error) {
	digestPos := calcDigestPosition(data, offset)
	buf := new(bytes.Buffer)
	buf.Write(data[:digestPos])             // random-data-1
	buf.Write(data[digestPos+_SizeDigest:]) // random-data-2
	return hmacSha256(buf.Bytes(), secret)  // digest-data
}

// KEY(764bytes):
//		random-data -> (offset)bytes
//		key-data    -> 128bytes
//		random-data -> (764-offset-128-4)bytes
//		offset      -> 4bytes
func generatePublicKey(data []byte, offset int, secret []byte) ([]byte, error) {
	// TODO: generate public key
	return nil, nil
}
