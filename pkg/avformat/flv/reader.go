package flv

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// Reader .
type Reader struct {
	r io.Reader
}

// ReadFlvHeader .
func (fr *Reader) ReadFlvHeader() ([]byte, error) {
	flvHeader := make([]byte, 9)
	if _, err := fr.r.Read(flvHeader); err != nil {
		return nil, err
	}
	// valid flv header
	if !bytes.Equal(FlvHeader, flvHeader) {
		return nil, fmt.Errorf("FLV: header invalid, %v", flvHeader)
	}
	return flvHeader, nil
}

// ReadPreTagSize0 always 0
func (fr *Reader) ReadPreTagSize0() error {
	var size uint32
	if err := binary.Read(fr.r, binary.BigEndian, &size); err != nil {
		return err
	}
	if size != 0 {
		return fmt.Errorf("FLV: first pre-tag size error, must be 0")
	}
	return nil
}

// ReadFlvSegment, format: tag-header(11bytes) + tag-data + pre-tag-size(4bytes)
func (fr *Reader) ReadFlvSegment() (*Tag, uint32, error) {
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
	buf = make([]byte, flvTag.TagHeader.DataSize)
	if _, err := io.ReadFull(fr.r, buf); err != nil {
		return nil, 0, err
	}
	flvTag.TagData = buf

	// previous tag size
	var size uint32
	if err := binary.Read(fr.r, binary.BigEndian, &size); err != nil {
		return nil, 0, err
	}

	return flvTag, size, nil
}
