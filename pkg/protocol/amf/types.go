package amf

import "io"

const (
	amf0 = uint(0)
	amf3 = uint(3)
)

// AMF-0 Data Types
const (
	amf0Number        = 0x00
	amf0Boolean       = 0x01
	amf0String        = 0x02
	amf0Object        = 0x03
	amf0Movieclip     = 0x04 // reserved, not supported
	amf0Null          = 0x05
	amf0Undefined     = 0x06
	amf0Reference     = 0x07
	amf0EcmaArray     = 0x08
	amf0ObjectEnd     = 0x09
	amf0StrictArray   = 0x0A
	amf0Date          = 0x0B
	amf0LongString    = 0x0C
	amf0Unsupported   = 0x0D
	amf0Recordset     = 0x0E // reserved, not supported
	amf0XmlDocument   = 0x0F
	amf0TypedObject   = 0x10
	amf0AvmplusObject = 0x11
)

// AMF-3 Data Types
const (
	amf3Undefined    = 0x00
	amf3Null         = 0x01
	amf3False        = 0x02
	amf3True         = 0x03
	amf3Integer      = 0x04
	amf3Double       = 0x05
	amf3String       = 0x06
	amf3Xmldoc       = 0x07
	amf3Date         = 0x08
	amf3Array        = 0x09
	amf3Object       = 0x0A
	amf3Xml          = 0x0B
	amf3ByteArray    = 0x0C
	amf3VectorInt    = 0x0D
	amf3VectorUint   = 0x0E
	amf3VectorDouble = 0x0F
	amf3VectorObject = 0x10
	amf3Dictionary   = 0x11
)

// Writer .
type Writer interface {
	io.Writer
	io.ByteWriter
}

// Reader .
type Reader interface {
	io.Reader
	io.ByteReader
}

// AMF0 .
type AMF0 struct{}

// AMF3 .
type AMF3 struct{}
