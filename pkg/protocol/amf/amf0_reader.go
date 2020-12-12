package amf

import (
	"encoding/binary"
	"fmt"
)

// ReadFrom .
func (amf *AMF0) ReadFrom(r Reader) (interface{}, error) {
	marker, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("amf0: error to read amf0 marker, %s", err)
	}

	switch marker {
	case amf0Number:
		return amf.ReadNumber(r)
	case amf0Boolean:
		return amf.ReadBoolean(r)
	case amf0String:
		return amf.ReadString(r)
	case amf0Object:
		return amf.ReadObject(r)
	case amf0Movieclip:
		return nil, fmt.Errorf("amf0: unsupported type movieclip")
	case amf0Null:
		return amf.ReadNull(r)
	case amf0Undefined:
		return amf.ReadUndefined(r)
	case amf0Reference:
		return nil, fmt.Errorf("amf0: unsupported type reference")
	case amf0EcmaArray:
		return amf.ReadEcamArray(r)
	case amf0ObjectEnd:
		return nil, fmt.Errorf("amf0: error position type object end")
	case amf0StrictArray:
		return amf.ReadStrictArray(r)
	case amf0Date:
		return amf.ReadDate(r)
	case amf0LongString:
		return amf.ReadLongString(r)
	case amf0Unsupported:
		return amf.ReadUnsupported(r)
	case amf0Recordset:
		return nil, fmt.Errorf("amf0: unsupported type recordset")
	case amf0XmlDocument:
		return nil, fmt.Errorf("amf0: unsupported type xml document")
	case amf0TypedObject:
		return nil, fmt.Errorf("amf0: unsupported type type object")
	case amf0AvmplusObject:
		return nil, fmt.Errorf("amf0: unsupported type avm plus object")
	}
	return nil, fmt.Errorf("amf0: unsupported type %d", marker)
}

// ReadNumber .
//  - number-marker DOUBLE
func (amf *AMF0) ReadNumber(r Reader) (data float64, err error) {
	if err = binary.Read(r, binary.BigEndian, &data); err != nil {
		return float64(0), fmt.Errorf("amf0: error to read number, %s", err)
	}
	return
}

// ReadBoolean .
//  - boolean-marker U8
func (amf *AMF0) ReadBoolean(r Reader) (data bool, err error) {
	var b byte
	if b, err = r.ReadByte(); err != nil {
		return false, fmt.Errorf("amf0: error to read boolean, %s", err)
	}
	if b == 0x00 {
		return false, nil
	}
	if b == 0x01 {
		return true, nil
	}
	return false, fmt.Errorf("amf0: unexpected value %v for boolean", b)
}

// ReadString .
//  - string-marker UTF-8
func (amf *AMF0) ReadString(r Reader) (data string, err error) {
	var length uint16
	err = binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return "", fmt.Errorf("amf0: error to read string length, %s", err)
	}
	if length == 0 {
		return "", nil
	}

	var bytes = make([]byte, length)
	if _, err = r.Read(bytes); err != nil {
		return "", fmt.Errorf("amf0: error to read string value, %s", err)
	}
	return string(bytes), nil
}

// ReadObject .
// 	- object-marker *((UTF-8 value-type) | (UTF-8-empty object-end-marker))
func (amf *AMF0) ReadObject(r Reader) (data map[string]interface{}, err error) {
	object := make(map[string]interface{})

	for {
		key, err := amf.ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("amf0: error to read property of object, %s", err)
		}
		if key == "" {
			b, err := r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("amf0: error to read object end-marker, %s", err)
			}
			if b == amf0ObjectEnd {
				break
			} else {
				return nil, fmt.Errorf("amf0: expected object end-marker, %s", err)
			}
		}

		value, err := amf.ReadFrom(r)
		if err != nil {
			return nil, fmt.Errorf("amf0: error to read object value, %s", err)
		}
		object[key] = value
	}

	return object, nil
}

// ReadNull .
func (amf *AMF0) ReadNull(r Reader) (data interface{}, err error) {
	return nil, nil
}

// ReadUndefined .
func (amf *AMF0) ReadUndefined(r Reader) (data interface{}, err error) {
	return nil, nil
}

// ReadEcamArray .
//  - associative-count *((UTF-8 value-type) | (UTF-8-empty object-end-marker))
func (amf *AMF0) ReadEcamArray(r Reader) (data map[string]interface{}, err error) {
	var length uint32
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("amf0: error to read ecam array length, %s", err)
	}
	if length == 0 {
		return nil, nil
	}

	if data, err = amf.ReadObject(r); err != nil {
		return nil, fmt.Errorf("amf0: error to read ecma array element, %s", err)
	}

	return
}

// ReadStrictArray .
//  - array-count *(value-type)
func (amf *AMF0) ReadStrictArray(r Reader) (data []interface{}, err error) {
	var length uint32
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("amf0: error to read strict array length, %s", err)
	}
	if length == 0 {
		return nil, nil
	}

	data = make([]interface{}, length)
	for idx := uint32(0); idx < length; idx++ {
		if data[idx], err = amf.ReadDate(r); err != nil {
			return nil, fmt.Errorf("amf0: error to read strict array element, %s", err)
		}
	}

	return
}

// ReadDate .
//  - date-marker DOUBLE time-zone
func (amf *AMF0) ReadDate(r Reader) (data float64, err error) {
	if data, err = amf.ReadNumber(r); err != nil {
		return float64(0), fmt.Errorf("amf0: error to read date value, %s", err)
	}

	length := make([]byte, 2)
	if _, err = r.Read(length); err != nil {
		return float64(0), fmt.Errorf("amf0: error to timezone of date, %s", err)
	}
	return
}

// ReadLongString .
//  - long-string-marker UTF-8-long
func (amf *AMF0) ReadLongString(r Reader) (data string, err error) {
	var length uint32
	err = binary.Read(r, binary.BigEndian, &length)
	if err != nil {
		return "", fmt.Errorf("amf0: error to read long string length, %s", err)
	}
	if length == 0 {
		return "", nil
	}

	var bytes = make([]byte, length)
	if _, err = r.Read(bytes); err != nil {
		return "", fmt.Errorf("amf0: error to read long string value, %s", err)
	}
	return string(bytes), nil
}

// ReadUnsupported .
func (amf *AMF0) ReadUnsupported(r Reader) (data interface{}, err error) {
	return nil, nil
}

// ReadXMLDocument .
func (amf *AMF0) ReadXMLDocument(r Reader) (data string, err error) {
	panic(fmt.Errorf("amf0: not support to read xml document value, %s", err))
}

// ReadTypeObject .
func (amf *AMF0) ReadTypeObject(r Reader) (data interface{}, err error) {
	panic(fmt.Errorf("amf0: not support to read movieclip value, %s", err))
}
