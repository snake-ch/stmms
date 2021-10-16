package flv

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Writer .
type Writer struct {
	app    string
	stream string
	w      io.Writer
}

// NewWriter .
func NewWriter(w io.Writer, app string, stream string) (*Writer, error) {
	fw := &Writer{
		app:    app,
		stream: stream,
		w:      w,
	}
	// flv header: 0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09
	if _, err := w.Write(FlvHeader); err != nil {
		return nil, err
	}
	// first pre-tag size always 0
	if err := binary.Write(w, binary.BigEndian, uint32(0)); err != nil {
		return nil, err
	}
	return fw, nil
}

// WriteTag write tag and previous tag size. ex. message from rtmp publisher
func (fw *Writer) WriteTag(tag *Tag) error {
	buf := new(bytes.Buffer)

	// tag type, 1byte
	buf.WriteByte(tag.TagHeader.TagType)

	// data size, 3bytes
	size := tag.TagHeader.DataSize
	buf.Write([]byte{byte(size >> 16), byte(size >> 8), byte(size)})

	// timestamp + timestamp extended, 4bytes
	timestamp := tag.TagHeader.Timestamp & 0xFFFFFF
	timestampExt := tag.TagHeader.Timestamp >> 24 & 0xFF
	buf.Write([]byte{byte(timestamp >> 16), byte(timestamp >> 8), byte(timestamp)})
	buf.WriteByte(byte(timestampExt))

	// stream id, always 0, 3bytes
	buf.Write([]byte{0x00, 0x00, 0x00})

	// tag-header and tag-data
	if _, err := fw.w.Write(buf.Bytes()); err != nil {
		return err
	}
	if _, err := fw.w.Write(tag.TagData); err != nil {
		return err
	}

	// previous tag-size
	pSize := make([]byte, 4)
	binary.BigEndian.PutUint32(pSize, tag.TagHeader.DataSize+11)
	if _, err := fw.w.Write(pSize); err != nil {
		return err
	}
	return nil
}

// WriteRawTag write flv-tag raw data direct, padding fixs data offset,
// ex. data read from flv file source or data from rtp or others
func (fw *Writer) WriteRawTag(rawTag []byte, padding uint32) error {
	// flv raw tag
	if _, err := fw.w.Write(rawTag); err != nil {
		return err
	}
	// previous tag size
	pSize := make([]byte, 4)
	binary.BigEndian.PutUint32(pSize, uint32(len(rawTag))+padding)
	if _, err := fw.w.Write(pSize); err != nil {
		return err
	}
	return nil
}
