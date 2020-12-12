package flv

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Reader .
type Reader struct {
	r         io.Reader
	flvHeader []byte
}

// ReadFlvHeader .
func (fr *Reader) ReadFlvHeader() ([]byte, error) {
	flvHeader := make([]byte, 9)
	if _, err := fr.r.Read(flvHeader); err != nil {
		return nil, err
	}
	// valid flv header
	if bytes.Compare(FlvHeader, flvHeader) != 0 {
		return nil, fmt.Errorf("flv header invalid, %v", flvHeader)
	}
	fr.flvHeader = flvHeader
	return flvHeader, nil
}

// ReadPreTagSize0 always 0
func (fr *Reader) ReadPreTagSize0() error {
	var size uint32
	if err := binary.Read(fr.r, binary.BigEndian, &size); err != nil {
		return err
	}
	if 0 != size {
		return fmt.Errorf("flv first pre-tag size err, should be 0")
	}
	return nil
}

// ReadFlvFragment .
// format: tag-header(11bytes) + tag-data + pre-tag-size(4bytes)
func (fr *Reader) ReadFlvFragment() (*Tag, uint32, error) {
	flvTag := &Tag{}

	// tag header
	buf := make([]byte, 11)
	if _, err := io.ReadFull(fr.r, buf); err != nil {
		return nil, 0, err
	}
	if err := flvTag.ParseTagHeader(buf); err != nil {
		return nil, 0, err
	}

	// tag data
	buf = make([]byte, flvTag.tagHeader.dataSize)
	if _, err := io.ReadFull(fr.r, buf); err != nil {
		return nil, 0, err
	}
	flvTag.tagData = buf

	// previous tag size
	var size uint32
	if err := binary.Read(fr.r, binary.BigEndian, &size); err != nil {
		return nil, 0, err
	}

	return flvTag, size, nil
}
