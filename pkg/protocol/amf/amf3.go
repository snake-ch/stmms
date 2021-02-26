package amf

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"
)

/*******************************
 * AMF3 Reader
 *******************************/

// ReadFrom .
func (amf *AMF3) ReadFrom(r Reader) (data interface{}, err error) {
	marker, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("amf3: error to read amf3 marker, %s", err)
	}

	switch marker {
	case _AMF3Undefined:
		return amf.ReadUndefined(r)
	case _AMF3Null:
		return amf.ReadNull(r)
	case _AMF3False:
		return amf.ReadFalse(r)
	case _AMF3True:
		return amf.ReadTrue(r)
	case _AMF3Integer:
		return amf.ReadInteger(r)
	case _AMF3Double:
		return amf.ReadDouble(r)
	case _AMF3String:
		return amf.ReadString(r)
	case _AMF3Xmldoc:
		return nil, fmt.Errorf("amf3: not support to read xml doc")
	case _AMF3Array:
		return amf.ReadArray(r)
	case _AMF3Object:
		return amf.ReadObject(r)
	case _AMF3Xml:
		return nil, fmt.Errorf("amf3: not support to read xml")
	case _AMF3ByteArray:
		return amf.ReadByteArray(r)
	case _AMF3VectorInt, _AMF3VectorUint, _AMF3VectorDouble, _AMF3VectorObject:
		return nil, fmt.Errorf("amf3: not support to read vector value")
	case _AMF3Dictionary:
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
	return struct{}{}, nil
}

// ReadNull .
//  - null-marker
func (amf *AMF3) ReadNull(r Reader) (data interface{}, err error) {
	return nil, nil
}

// ReadFalse .
//  - false-marker
func (amf *AMF3) ReadFalse(r Reader) (data bool, err error) {
	return false, nil
}

// ReadTrue .
//  - true-marker
func (amf *AMF3) ReadTrue(r Reader) (data bool, err error) {
	return true, nil
}

// ReadInteger .
//  - interger-marker U29
func (amf *AMF3) ReadInteger(r Reader) (data int32, err error) {
	u29, err := amf.ReadU29(r)
	if err != nil {
		return 0, fmt.Errorf("amf3: error to read integer, %s", err)
	}

	data = int32(u29)
	if data > 0xFFFFFFF {
		data = int32(u29 - 0x20000000)
	}
	return
}

// ReadDouble .
func (amf *AMF3) ReadDouble(r Reader) (data float64, err error) {
	if err = binary.Read(r, binary.BigEndian, &data); err != nil {
		return 0, fmt.Errorf("amf3: error to read double, %s", err)
	}
	return
}

// ReadString .
//  - string-marker UTF-8-vr
func (amf *AMF3) ReadString(r Reader) (data string, err error) {
	return amf.ReadUTF8(r)
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
				prop, err := amf.ReadString(r)
				if err != nil {
					return nil, fmt.Errorf("amf3: error to read object dynamic member property")
				}
				// dynamic member ends with an empty string
				if prop == "" {
					break
				}
				// value
				value, err := amf.ReadFrom(r)
				if err != nil {
					return nil, fmt.Errorf("amf3: error to read object dynamic member value")
				}
				data[prop] = value
			}
		}
	default:
		return nil, fmt.Errorf("amf3: unsupported traits object")
	}
	return
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

/*******************************
 * AMF3 Writer
 *******************************/

// WriteTo .
func (amf *AMF3) WriteTo(w Writer, val interface{}) (n int, err error) {
	if val == nil {
		return amf.WriteNull(w)
	}

	v := reflect.ValueOf(val)
	if !v.IsValid() {
		return amf.WriteNull(w)
	}

	switch v.Kind() {
	case reflect.String:
		return amf.WriteString(w, v.String())
	case reflect.Bool:
		if v.Bool() {
			return amf.WriteTrue(w)
		}
		return amf.WriteFalse(w)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		number := v.Int()
		if number >= 0 && n <= 0x1FFFFFFF {
			return amf.WriteInteger(w, uint32(number))
		}
		return amf.WriteDouble(w, float64(number))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		number := v.Float()
		if number <= 0x1FFFFFFF {
			return amf.WriteInteger(w, uint32(number))
		}
		return amf.WriteDouble(w, float64(number))
	case reflect.Float32, reflect.Float64:
		return amf.WriteDouble(w, v.Float())
	case reflect.Array, reflect.Slice:
		array := make([]interface{}, v.Len())
		for idx := 0; idx < v.Len(); idx++ {
			array[idx] = v.Index(int(idx)).Interface()
		}
		return amf.WriteArray(w, array)
	case reflect.Map:
		if m, ok := val.(map[string]interface{}); ok {
			return amf.WriteObject(w, m)
		}
		return 0, fmt.Errorf("amf3: error to write object from map")
	}

	if t, ok := val.(time.Time); ok {
		return amf.WriteDate(w, t)
	}

	return 0, fmt.Errorf("amf3: unsupported type %s", v.Type())
}

// WriteU29 .
func (amf *AMF3) WriteU29(w Writer, val uint32) (n int, err error) {
	if val <= 0x0000007F {
		if err = w.WriteByte(byte(val)); err != nil {
			return
		}
		n = n + 1
	} else if n <= 0x00003FFF {
		return w.Write([]byte{byte(n>>7 | 0x80), byte(n & 0x7F)})
	} else if n <= 0x001FFFFF {
		return w.Write([]byte{byte(n>>14 | 0x80), byte(n>>7&0x7F | 0x80), byte(n & 0x7F)})
	} else if n <= 0x1FFFFFFF {
		return w.Write([]byte{byte(n>>22 | 0x80), byte(n>>15&0x7F | 0x80), byte(n>>8&0x7F | 0x80), byte(n)})
	}
	return 0, fmt.Errorf("amf3: write U29 with value %d (out of range)", val)
}

// WriteUTF8 .
//  - U29S-ref
//  ✔ U29-value *(UTF8-char)
func (amf *AMF3) WriteUTF8(w Writer, val string) (n int, err error) {
	// U29-value
	number := 0
	if number, err = amf.WriteU29(w, uint32((len(val)<<1)|0x01)); err != nil {
		return 0, fmt.Errorf("amf3: error to write string length, %s", err)
	}
	n = n + number

	// *(UTF8-char)
	if number, err = w.Write([]byte(val)); err != nil {
		return n, fmt.Errorf("amf3: error to write string value, %s", err)
	}
	return n + number, nil
}

// WriteUndefined .
//  - undefined-marker
func (amf *AMF3) WriteUndefined(w Writer) (n int, err error) {
	if err = w.WriteByte(_AMF3Undefined); err != nil {
		return n, fmt.Errorf("amf3: error to write undefined marker, %s", err)
	}
	return n + 1, nil
}

// WriteNull .
//  - null-marker
func (amf *AMF3) WriteNull(w Writer) (n int, err error) {
	if err = w.WriteByte(_AMF3Null); err != nil {
		return n, fmt.Errorf("amf3: error to write null marker, %s", err)
	}
	return n + 1, nil
}

// WriteFalse .
//  - false-marker
func (amf *AMF3) WriteFalse(w Writer) (n int, err error) {
	if err = w.WriteByte(_AMF3False); err != nil {
		return n, fmt.Errorf("amf3: error to write false marker, %s", err)
	}
	return n + 1, nil
}

// WriteTrue .
//  - true-marker
func (amf *AMF3) WriteTrue(w Writer) (n int, err error) {
	if err = w.WriteByte(_AMF3True); err != nil {
		return n, fmt.Errorf("amf3: error to write true marker, %s", err)
	}
	return n + 1, nil
}

// WriteInteger .
//  - interger-marker U29
func (amf *AMF3) WriteInteger(w Writer, val uint32) (n int, err error) {
	// interger-marker
	if err = w.WriteByte(_AMF3Integer); err != nil {
		return n, fmt.Errorf("amf3: error to write integer marker, %s", err)
	}
	n = n + 1

	// U29
	number := 0
	if number, err = amf.WriteU29(w, val); err != nil {
		return n, fmt.Errorf("amf3: error to write integer value, %s", err)
	}
	return n + number, nil
}

// WriteDouble .
//  - double-marker Double
func (amf *AMF3) WriteDouble(w Writer, val float64) (n int, err error) {
	// double-marker
	if err = w.WriteByte(_AMF3Double); err != nil {
		return n, fmt.Errorf("amf3: error to write double marker, %s", err)
	}
	n = n + 1

	// Double
	if err = binary.Write(w, binary.BigEndian, val); err != nil {
		return n, fmt.Errorf("amf3: error to write double value, %s", err)
	}
	return n + 8, nil
}

// WriteString .
//  - string-marker UTF-8-vr
func (amf *AMF3) WriteString(w Writer, val string) (n int, err error) {
	// string-marker
	if err = w.WriteByte(_AMF3String); err != nil {
		return n, fmt.Errorf("amf3: error to write string marker, %s", err)
	}
	n = n + 1

	// UTF-8-vr
	number := 0
	if number, err = amf.WriteUTF8(w, val); err != nil {
		return n, fmt.Errorf("amf3: error to write string length and value, %s", err)
	}
	return n + number, nil
}

// WriteDate .
//  - date-marker U29O-ref
//  ✔ date-marker U29D-value date-time
func (amf *AMF3) WriteDate(w Writer, val time.Time) (n int, err error) {
	// date-marker
	if err = w.WriteByte(_AMF3Date); err != nil {
		return n, fmt.Errorf("amf3: error to write date marker, %s", err)
	}
	n = n + 1

	// U29D-value
	if err = w.WriteByte(0x01); err != nil {
		return n, fmt.Errorf("amf3: error to write 0x01 for Date, %s", err)
	}
	n = n + 1

	// date-time
	timestamp := val.Unix() * 1000
	if err = binary.Write(w, binary.BigEndian, float64(timestamp)); err != nil {
		return n, fmt.Errorf("amf3: error to write date value, %s", err)
	}
	return n + 8, nil
}

// WriteArray .
//  - array-marker U29O-ref
//  ✔ array-marker U29A-value UTF-8-empty *(value-type)
//  - array-marker U29A-value *(assoc-value) UTF-8-empty *(value-type)
func (amf *AMF3) WriteArray(w Writer, val []interface{}) (n int, err error) {
	// array-marker
	if err = w.WriteByte(_AMF3Array); err != nil {
		return n, fmt.Errorf("amf3: error to write array marker, %s", err)
	}
	n = n + 1

	// U29A-value
	number := 0
	if number, err = amf.WriteU29(w, uint32((len(val)<<1)|0x01)); err != nil {
		return 0, fmt.Errorf("amf3: error to write array length, %s", err)
	}
	n = n + number

	// UTF-8-empty
	if number, err = amf.WriteUTF8(w, ""); err != nil {
		return n, fmt.Errorf("amf3: error to write array empty string, %s", err)
	}
	n = n + number

	// *(value-type)
	for _, value := range val {
		if number, err = amf.WriteTo(w, value); err != nil {
			return n, fmt.Errorf("amf3: error to write dense array element, %s", err)
		}
		n = n + number
	}

	return
}

// WriteObject .
//  - object-marker U29O-ref
//  - object-marker U29O-traits-ref
//  - object-marker U29O-traits-ext class-name *(U8)
//  ✔ object-marker U29O-traits class-name *(UTF-8-vr) *(value-type) *(dynamic-member)
func (amf *AMF3) WriteObject(w Writer, val map[string]interface{}) (n int, err error) {
	// object marker
	if err = w.WriteByte(_AMF3Object); err != nil {
		return n, fmt.Errorf("amf3: error to write object marker, %s", err)
	}
	n = n + 1

	// U29O-traits(only-dynamic)
	if err = w.WriteByte(0x0b); err != nil {
		return n, fmt.Errorf("amf3: error to write object traits flags, %s", err)
	}
	n = n + 1

	// class-name(empty)
	number := 0
	if number, err = amf.WriteUTF8(w, ""); err != nil {
		return n, fmt.Errorf("amf3: error to write object empty string class name, %s", err)
	}
	n = n + number

	// *(dynamic-member)
	for key, value := range val {
		if number, err = amf.WriteUTF8(w, key); err != nil {
			return n, fmt.Errorf("amf3: error to write dynamic member key of Object, %s", err)
		}
		n = n + number

		if number, err = amf.WriteTo(w, value); err != nil {
			return n, fmt.Errorf("amf3: error to write dynamic member value of Object, %s", err)
		}
		n = n + number
	}

	if number, err = amf.WriteUTF8(w, ""); err != nil {
		return n, fmt.Errorf("amf3: error to write dynamic member end marker of Object, %s", err)
	}
	return n + number, nil
}

// WriteBytearray .
//  - bytearray-marker U29O-ref
//  ✔ bytearray-marker U29B-value *(U8)
func (amf *AMF3) WriteBytearray(w Writer, val []byte) (n int, err error) {
	// bytearray marker
	if err = w.WriteByte(_AMF3ByteArray); err != nil {
		return n, fmt.Errorf("amf3: error to write bytearray marker, %s", err)
	}
	n = n + 1

	// U29B-value
	number := 0
	if number, err = amf.WriteU29(w, uint32((len(val)<<1)|0x01)); err != nil {
		return 0, fmt.Errorf("amf3: error to write bytearray length, %s", err)
	}
	n = n + number

	// *(U8)
	if number, err = w.Write(val); err != nil {
		return n, fmt.Errorf("amf3: error to write bytearray value, %s", err)
	}
	return n + number, nil
}
