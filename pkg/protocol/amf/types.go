package amf

import "io"

// AMF-0 Types
const (
	_AMF0Number        = 0x00
	_AMF0Boolean       = 0x01
	_AMF0String        = 0x02
	_AMF0Object        = 0x03
	_AMF0Movieclip     = 0x04 // reserved, not supported
	_AMF0Null          = 0x05
	_AMF0Undefined     = 0x06
	_AMF0Reference     = 0x07
	_AMF0EcmaArray     = 0x08
	_AMF0ObjectEnd     = 0x09
	_AMF0StrictArray   = 0x0A
	_AMF0Date          = 0x0B
	_AMF0LongString    = 0x0C
	_AMF0Unsupported   = 0x0D
	_AMF0Recordset     = 0x0E // reserved, not supported
	_AMF0XmlDocument   = 0x0F
	_AMF0TypedObject   = 0x10
	_AMF0AvmplusObject = 0x11
)

// AMF-3 Types
const (
	_AMF3Undefined    = 0x00
	_AMF3Null         = 0x01
	_AMF3False        = 0x02
	_AMF3True         = 0x03
	_AMF3Integer      = 0x04
	_AMF3Double       = 0x05
	_AMF3String       = 0x06
	_AMF3Xmldoc       = 0x07
	_AMF3Date         = 0x08
	_AMF3Array        = 0x09
	_AMF3Object       = 0x0A
	_AMF3Xml          = 0x0B
	_AMF3ByteArray    = 0x0C
	_AMF3VectorInt    = 0x0D
	_AMF3VectorUint   = 0x0E
	_AMF3VectorDouble = 0x0F
	_AMF3VectorObject = 0x10
	_AMF3Dictionary   = 0x11
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
