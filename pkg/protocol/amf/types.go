package amf

import "io"

// AMF-0 Types
const (
	AMF0Number        = 0x00
	AMF0Boolean       = 0x01
	AMF0String        = 0x02
	AMF0Object        = 0x03
	AMF0Movieclip     = 0x04 // reserved, not supported
	AMF0Null          = 0x05
	AMF0Undefined     = 0x06
	AMF0Reference     = 0x07
	AMF0EcmaArray     = 0x08
	AMF0ObjectEnd     = 0x09
	AMF0StrictArray   = 0x0A
	AMF0Date          = 0x0B
	AMF0LongString    = 0x0C
	AMF0Unsupported   = 0x0D
	AMF0Recordset     = 0x0E // reserved, not supported
	AMF0XmlDocument   = 0x0F
	AMF0TypedObject   = 0x10
	AMF0AvmplusObject = 0x11
)

// AMF-3 Types
const (
	AMF3Undefined    = 0x00
	AMF3Null         = 0x01
	AMF3False        = 0x02
	AMF3True         = 0x03
	AMF3Integer      = 0x04
	AMF3Double       = 0x05
	AMF3String       = 0x06
	AMF3Xmldoc       = 0x07
	AMF3Date         = 0x08
	AMF3Array        = 0x09
	AMF3Object       = 0x0A
	AMF3Xml          = 0x0B
	AMF3ByteArray    = 0x0C
	AMF3VectorInt    = 0x0D
	AMF3VectorUint   = 0x0E
	AMF3VectorDouble = 0x0F
	AMF3VectorObject = 0x10
	AMF3Dictionary   = 0x11
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
