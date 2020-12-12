package amf

import (
	"encoding/binary"
	"fmt"
	"time"
)

// ReadFrom .
func (amf *AMF3) ReadFrom(r Reader) (data interface{}, err error) {
	marker, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("amf3: error to read amf3 marker, %s", err)
	}

	switch marker {
	case amf3Undefined:
		return amf.ReadUndefined(r)
	case amf3Null:
		return amf.ReadNull(r)
	case amf3False:
		return amf.ReadFalse(r)
	case amf3True:
		return amf.ReadTrue(r)
	case amf3Integer:
		return amf.ReadInteger(r)
	case amf3Double:
		return amf.ReadDouble(r)
	case amf3String:
		return amf.ReadString(r)
	case amf3Xmldoc:
		return amf.ReadXMLDoc(r)
	case amf3Array:
		return amf.ReadArray(r)
	case amf3Object:
		return amf.ReadObject(r)
	case amf3Xml:
		return amf.ReadXML(r)
	case amf3ByteArray:
		return amf.ReadByteArray(r)
	case amf3VectorInt, amf3VectorUint, amf3VectorDouble, amf3VectorObject:
		return nil, fmt.Errorf("amf3: not support to read vector value")
	case amf3Dictionary:
		return nil, fmt.Errorf("amf3: not support to read dictionary value")
	}
	return nil, fmt.Errorf("amf3: unsupported type %d", marker)
}

// ReadU29 .
// 	- 0xxxxxxx
// 	- 1xxxxxxx 0xxxxxxx
// 	- 1xxxxxxx 1xxxxxxx 0xxxxxxx
// 	- 1xxxxxxx 1xxxxxxx 1xxxxxxx xxxxxxxx
func (amf *AMF3) ReadU29(r Reader) (data uint32, err error) {
	var b byte

	// the first 3 bytes
	for idx := 0; idx < 3; idx++ {
		if b, err = r.ReadByte(); err != nil {
			return
		}
		data = (data << 7) + uint32(b&0x7F)
		if (b & 0x80) == 0 {
			return data, fmt.Errorf("error to read first 3 bytes of U29, %s", err)
		}
	}

	// 4th byte
	if b, err = r.ReadByte(); err != nil {
		return data, fmt.Errorf("error to read 4th byte of U29, %s", err)
	}
	return (data << 7) + uint32(b), nil
}

// ReadUTF8 .
//  - U29S-ref
//  ✔ U29-value *(UTF8-char)
func (amf *AMF3) ReadUTF8(r Reader) (data string, err error) {
	// U29-value
	var length uint32
	if length, err = amf.ReadU29(r); err != nil {
		return "", fmt.Errorf("amf3: error to read string length, %s", err)
	}

	if length&0x01 != 0x01 {
		return "", fmt.Errorf("amf3: not support to read string reference")
	}

	length = length >> 1
	if length == 0 {
		return "", nil
	}

	// *(UTF8-char)
	buf := make([]byte, length)
	if _, err = r.Read(buf); err != nil {
		return "", fmt.Errorf("amf3: unable to read string, %s", err)
	}

	return string(buf), nil
}

// ReadUndefined .
//  - undefined-marker
func (amf *AMF3) ReadUndefined(r Reader) (data interface{}, err error) {
	panic(fmt.Errorf("amf3: not support read undefined value, %s", err))
}

// ReadNull .
//  - null-marker
func (amf *AMF3) ReadNull(r Reader) (data interface{}, err error) {
	panic(fmt.Errorf("amf3: not support read null value, %s", err))
}

// ReadFalse .
//  - false-marker
func (amf *AMF3) ReadFalse(r Reader) (data bool, err error) {
	panic(fmt.Errorf("amf3: not support read false value, %s", err))
}

// ReadTrue .
//  - true-marker
func (amf *AMF3) ReadTrue(r Reader) (data bool, err error) {
	panic(fmt.Errorf("amf3: not support read true value, %s", err))
}

// ReadInteger .
//  - interger-marker U29
func (amf *AMF3) ReadInteger(r Reader) (data uint32, err error) {
	return amf.ReadU29(r)
}

// ReadDouble .
func (amf *AMF3) ReadDouble(r Reader) (data float64, err error) {
	if err = binary.Read(r, binary.BigEndian, &data); err != nil {
		return float64(0), fmt.Errorf("amf3: error to read double, %s", err)
	}
	return
}

// ReadString .
//  - string-marker UTF-8-vr
func (amf *AMF3) ReadString(r Reader) (data string, err error) {
	return amf.ReadUTF8(r)
}

// ReadXMLDoc .
func (amf *AMF3) ReadXMLDoc(r Reader) (data interface{}, err error) {
	panic(fmt.Errorf("amf3: not support to read xml document value, %s", err))
}

// ReadDate .
//  - date-marker U29O-ref
//  ✔ date-marker U29D-value date-time
func (amf *AMF3) ReadDate(r Reader) (data time.Time, err error) {
	// U29D-value
	if _, err = amf.ReadU29(r); err != nil {
		return time.Time{}, fmt.Errorf("error to read date flags, %s", err)
	}

	// date-time
	var datetime float64
	if err = binary.Read(r, binary.BigEndian, &datetime); err != nil {
		return time.Time{}, fmt.Errorf("amf3: error to read date value, %s", err)
	}

	data = time.Unix(int64(datetime/1000), 0).UTC()
	return
}

// ReadArray .
//  - array-marker U29O-ref
//  ✔ array-marker U29A-value UTF-8-empty *(value-type)
//  - array-marker U29A-value *(assoc-value) UTF-8-empty *(value-type)
func (amf *AMF3) ReadArray(r Reader) (data []interface{}, err error) {
	// U29A-value
	var length uint32
	if length, err = amf.ReadU29(r); err != nil {
		return nil, fmt.Errorf("error to read byte array length, %s", err)
	}
	if length&0x01 != 0x01 {
		return nil, fmt.Errorf("amf3: not support to read byte array reference")
	}
	length = length >> 1
	if length == 0 {
		return nil, nil
	}

	// UTF-8-empty
	key, err := amf.ReadString(r)
	if err != nil {
		return nil, fmt.Errorf("amf3: error to read array empty string flag")
	}
	if key != "" {
		return nil, fmt.Errorf("amf3: not support to read associative portion of array")
	}

	// *(value-type)
	for idx := uint32(0); idx < length; idx++ {
		element, err := amf.ReadFrom(r)
		if err != nil {
			return data, fmt.Errorf("amf3: error to read array element, %s", err)
		}
		data = append(data, element)
	}
	return
}

// ReadObject .
//  - object-marker U29O-ref
//  - object-marker U29O-traits-ref
//  - object-marker U29O-traits-ext class-name *(U8)
//  ✔ object-marker U29O-traits class-name *(UTF-8-vr) *(value-type) *(dynamic-member)
//
//  current supported: object-marker U29O-traits class-name *(dynamic-member)
func (amf *AMF3) ReadObject(r Reader) (data map[string]interface{}, err error) {
	var traits uint32
	if traits, err = amf.ReadU29(r); err != nil {
		return nil, fmt.Errorf("error to read byte object traits, %s", err)
	}

	switch traits & 0x0F {
	case 0x00: // object reference
		return nil, fmt.Errorf("amf3: not support to read object reference")
	case 0x01: // traits reference
		return nil, fmt.Errorf("amf3: not support to read traits reference")
	case 0x07: // traits externalizable
		return nil, fmt.Errorf("amf3: not support to read traits externalizable")
	case 0x03, 0x0B: // sealed and dynamic object members
		// class-name
		_, err := amf.ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("amf3: error read object empty class name")
		}

		// sealed members(properties and values)
		if traits&0x0F == 0x03 {
			return nil, fmt.Errorf("amf3: not support to read object sealed members")
		}

		// dynamic-members
		if traits&0x0F == 0x0B {
			for {
				// property
				property, err := amf.ReadString(r)
				if err != nil {
					return nil, fmt.Errorf("amf3: error to read object dynamic member property")
				}
				// dynamic member ends with an empty string
				if property == "" {
					break
				}
				// value
				value, err := amf.ReadFrom(r)
				if err != nil {
					return nil, fmt.Errorf("amf3: error to read object dynamic member value")
				}
				data[property] = value
			}
		}
	default:
		return nil, fmt.Errorf("amf3: unsupported traits object")
	}
	return
}

// ReadXML .
func (amf *AMF3) ReadXML(r Reader) (data interface{}, err error) {
	panic(fmt.Errorf("amf3: not support to read xml value, %s", err))
}

// ReadByteArray .
//  - bytearray-marker U29O-ref
//  ✔ bytearray-marker U29B-value *(U8)
func (amf *AMF3) ReadByteArray(r Reader) (data []byte, err error) {
	// U29B-value
	var length uint32
	if length, err = amf.ReadU29(r); err != nil {
		return nil, fmt.Errorf("error to read byte array length, %s", err)
	}

	if length&0x01 != 0x01 {
		return nil, fmt.Errorf("amf3: not support to read byte array reference")
	}

	length = length >> 1
	if length == 0 {
		return nil, nil
	}

	// *(U8)
	data = make([]byte, length)
	if _, err = r.Read(data); err != nil {
		return nil, fmt.Errorf("amf3: unable to read byte array value, %s", err)
	}
	return
}

// ReadVertor .
func (amf *AMF3) ReadVertor(r Reader) (data interface{}, err error) {
	panic(fmt.Errorf("amf3: not support to read vector value, %s", err))
}

// ReadDictionary .
func (amf *AMF3) ReadDictionary(r Reader) (data interface{}, err error) {
	panic(fmt.Errorf("amf3: not support to read dictionary value, %s", err))
}
