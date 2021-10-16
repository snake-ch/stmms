package amf

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

/*******************************
 * AMF0 Reader
 *******************************/

// ReadFrom .
func (amf *AMF0) ReadFrom(r Reader) (interface{}, error) {
	marker, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("amf0: error to read amf0 marker, %s", err)
	}

	switch marker {
	case AMF0Number:
		return amf.ReadNumber(r)
	case AMF0Boolean:
		return amf.ReadBoolean(r)
	case AMF0String:
		return amf.ReadString(r)
	case AMF0Object:
		return amf.ReadObject(r)
	case AMF0Movieclip:
		return nil, fmt.Errorf("amf0: unsupported type movieclip")
	case AMF0Null:
		return amf.ReadNull(r)
	case AMF0Undefined:
		return amf.ReadUndefined(r)
	case AMF0Reference:
		return nil, fmt.Errorf("amf0: unsupported type reference")
	case AMF0EcmaArray:
		return amf.ReadEcmaArray(r)
	case AMF0ObjectEnd:
		return nil, fmt.Errorf("amf0: unsupported object end")
	case AMF0StrictArray:
		return amf.ReadStrictArray(r)
	case AMF0Date:
		return amf.ReadDate(r)
	case AMF0LongString:
		return amf.ReadLongString(r)
	case AMF0Unsupported:
		return amf.ReadUnsupported(r)
	case AMF0Recordset:
		return nil, fmt.Errorf("amf0: unsupported type recordset")
	case AMF0XmlDocument:
		return nil, fmt.Errorf("amf0: unsupported type xml document")
	case AMF0TypedObject:
		return nil, fmt.Errorf("amf0: unsupported type type object")
	case AMF0AvmplusObject:
		return nil, fmt.Errorf("amf0: unsupported type avm plus object")
	}
	return nil, fmt.Errorf("amf0: unsupported type %d", marker)
}

// ReadNumber .
//  - number-marker DOUBLE
func (amf *AMF0) ReadNumber(r Reader) (data float64, err error) {
	if err = binary.Read(r, binary.BigEndian, &data); err != nil {
		return 0, fmt.Errorf("amf0: error to read number, %s", err)
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
	obj := make(map[string]interface{})

	for {
		prop, err := amf.ReadString(r)
		if err != nil {
			return nil, fmt.Errorf("amf0: error to read property of object, %s", err)
		}
		if prop == "" {
			b, err := r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("amf0: error to read object end-marker, %s", err)
			}
			if b != AMF0ObjectEnd {
				return nil, fmt.Errorf("amf0: expected object end-marker, %s", err)
			}
			break
		}

		value, err := amf.ReadFrom(r)
		if err != nil {
			return nil, fmt.Errorf("amf0: error to read object value, %s", err)
		}
		obj[prop] = value
	}

	return obj, nil
}

// ReadNull .
func (amf *AMF0) ReadNull(r Reader) (data interface{}, err error) {
	return nil, nil
}

// ReadUndefined .
func (amf *AMF0) ReadUndefined(r Reader) (data interface{}, err error) {
	return struct{}{}, nil
}

// ReadEcmaArray .
//  - associative-count *((UTF-8 value-type) | (UTF-8-empty object-end-marker))
func (amf *AMF0) ReadEcmaArray(r Reader) (data map[string]interface{}, err error) {
	var length uint32
	if err = binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("amf0: error to read ecma array length, %s", err)
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
		if data[idx], err = amf.ReadFrom(r); err != nil {
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

	// reserved, should be 0x0000
	timezone := make([]byte, 2)
	if _, err = r.Read(timezone); err != nil {
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

/*******************************
 * AMF0 Writer
 *******************************/

// WriteTo .
func (amf *AMF0) WriteTo(w Writer, val interface{}) (n int, err error) {
	if val == nil {
		return amf.WriteNull(w)
	}

	v := reflect.ValueOf(val)
	if !v.IsValid() {
		return amf.WriteNull(w)
	}

	switch v.Kind() {
	case reflect.String:
		if len(v.String()) <= 0xFFFF {
			return amf.WriteString(w, v.String())
		}
		return amf.WriteLongString(w, v.String())
	case reflect.Bool:
		return amf.WriteBoolean(w, v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return amf.WriteNumber(w, float64(v.Int()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return amf.WriteNumber(w, float64(v.Uint()))
	case reflect.Float32, reflect.Float64:
		return amf.WriteNumber(w, v.Float())
	case reflect.Array, reflect.Slice:
		array := make([]interface{}, v.Len())
		for idx := 0; idx < v.Len(); idx++ {
			array[idx] = v.Index(int(idx)).Interface()
		}
		return amf.WriteStrictArray(w, array)
	case reflect.Map:
		if m, ok := val.(map[string]interface{}); ok {
			return amf.WriteObject(w, m)
		}
		return 0, fmt.Errorf("amf0: error to write object from map")
	case reflect.Ptr:
		if v.IsNil() || !v.IsValid() {
			return amf.WriteNull(w)
		}
		return amf.WriteTo(w, v.Elem().Interface())
	}

	return 0, fmt.Errorf("amf0: unsupported type %s", v.Type())
}

// WriteNumber .
//  - number-marker DOUBLE
func (amf *AMF0) WriteNumber(w Writer, val float64) (n int, err error) {
	// number-marker
	if err = w.WriteByte(AMF0Number); err != nil {
		return n, fmt.Errorf("amf0: error to write number marker, %s", err)
	}
	n = n + 1

	// DOUBLE
	if err = binary.Write(w, binary.BigEndian, val); err != nil {
		return n, fmt.Errorf("amf0: error to write double value, %s", err)
	}
	return n + 8, nil
}

// WriteBoolean .
//  - boolean-marker U8
func (amf *AMF0) WriteBoolean(w Writer, val bool) (n int, err error) {
	// boolean-marker
	if err = w.WriteByte(AMF0Boolean); err != nil {
		return n, fmt.Errorf("amf0: error to write boolean marker, %s", err)
	}
	n = n + 1

	// U8(=0 is false, <0 or >0 is true)
	if val {
		err = w.WriteByte(0x01)
	} else {
		err = w.WriteByte(0x00)
	}
	if err != nil {
		return n, fmt.Errorf("amf0: error to write boolean value, %s", err)
	}
	return n + 1, nil
}

// WriteString .
//  - string-marker UTF-8
func (amf *AMF0) WriteString(w Writer, val string) (n int, err error) {
	// string-marker
	if err = w.WriteByte(AMF0String); err != nil {
		return n, fmt.Errorf("amf0: error to write string marker, %s", err)
	}
	n = n + 1

	// U16
	if err = binary.Write(w, binary.BigEndian, uint16(len(val))); err != nil {
		return n, fmt.Errorf("amf0: error to write string length, %s", err)
	}
	n = n + 2

	// *(UTF8-char)
	if _, err = w.Write([]byte(val)); err != nil {
		return n, fmt.Errorf("amf0: error to write string value, %s", err)
	}
	return n + len(val), nil
}

// WriteObject .
// 	- object-marker *((UTF-8 value-type) | (UTF-8-empty object-end-marker))
func (amf *AMF0) WriteObject(w Writer, val map[string]interface{}) (n int, err error) {
	// object-marker
	if err = w.WriteByte(AMF0Object); err != nil {
		return n, fmt.Errorf("amf0: error to write object marker, %s", err)
	}
	n = n + 1

	number := 0
	for key, value := range val {
		// U16
		if err = binary.Write(w, binary.BigEndian, uint16(len(key))); err != nil {
			return n, fmt.Errorf("amf0: error to write object length, %s", err)
		}
		n = n + 2

		// *(UTF8-char)
		if number, err = w.Write([]byte(key)); err != nil {
			return n, fmt.Errorf("amf0: error to write key object property, %s", err)
		}
		n = n + number

		// value-type
		number, err = amf.WriteTo(w, value)
		if err != nil {
			return n, fmt.Errorf("amf0: error to write value of object property, %s", err)
		}
		n = n + number
	}

	// UTF-8-empty object-end-marker
	number, err = amf.WriteObjectEnd(w)
	return n + number, err
}

// WriteNull .
// 	- null-marker
func (amf *AMF0) WriteNull(w Writer) (n int, err error) {
	if err = w.WriteByte(AMF0Null); err != nil {
		return n, fmt.Errorf("amf0: error to write null marker, %s", err)
	}
	return 1, nil
}

// WriteUndefined .
// 	- undefined-marker
func (amf *AMF0) WriteUndefined(w Writer) (n int, err error) {
	if err = w.WriteByte(AMF0Undefined); err != nil {
		return n, fmt.Errorf("amf0: error to write undefined marker, %s", err)
	}
	return 1, nil
}

// WriteEcmaArray .
//  - associative-count *((UTF-8 value-type) | (UTF-8-empty object-end-marker))
func (amf *AMF0) WriteEcmaArray(w Writer, val map[string]interface{}) (n int, err error) {
	// ecma-array marker
	if err = w.WriteByte(AMF0EcmaArray); err != nil {
		return n, fmt.Errorf("amf0: error to write ecma-array marker, %s", err)
	}
	n = n + 1

	// associative-count
	if err = binary.Write(w, binary.BigEndian, uint32(len(val))); err != nil {
		return n, fmt.Errorf("amf0: error to write ecma-array length, %s", err)
	}
	n = n + 4

	number := 0
	for key, value := range val {
		// U16
		if err = binary.Write(w, binary.BigEndian, uint16(len(key))); err != nil {
			return n, fmt.Errorf("amf0: error to write index of ecma-array element, %s", err)
		}
		n = n + 2

		// *(UTF8-char)
		if number, err = w.Write([]byte(key)); err != nil {
			return n, fmt.Errorf("amf0: error to write key of ecma-array element, %s", err)
		}
		n = n + number

		// value-type
		number, err = amf.WriteTo(w, value)
		if err != nil {
			return n, fmt.Errorf("amf0: error to write value of ecma-array element, %s", err)
		}
		n = n + number
	}

	// UTF-8-empty object-end-marker
	number, err = amf.WriteObjectEnd(w)
	return n + number, err
}

// WriteObjectEnd .
//  - UTF-8-empty object-end-marker
func (amf *AMF0) WriteObjectEnd(w Writer) (n int, err error) {
	return w.Write([]byte{0x00, 0x00, AMF0ObjectEnd})
}

// WriteStrictArray .
//  - array-count *(value-type)
func (amf *AMF0) WriteStrictArray(w Writer, val []interface{}) (n int, err error) {
	// strict-array marker
	if err = w.WriteByte(AMF0StrictArray); err != nil {
		return n, fmt.Errorf("amf0: error to write strict-array marker, %s", err)
	}
	n = n + 1

	// array-count
	if err = binary.Write(w, binary.BigEndian, uint32(len(val))); err != nil {
		return n, fmt.Errorf("amf0: error to write strict-array length, %s", err)
	}
	n = n + 4

	// *(value-type)
	number := 0
	for _, value := range val {
		number, err = amf.WriteTo(w, value)
		if err != nil {
			return n, fmt.Errorf("amf0: error to write strict-array element, %s", err)
		}
		n = n + number
	}
	return
}

// WriteLongString .
//  - long-string-marker UTF-8-long
func (amf *AMF0) WriteLongString(w Writer, val string) (n int, err error) {
	// long-string-marker
	if err = w.WriteByte(AMF0LongString); err != nil {
		return n, fmt.Errorf("amf0: error to write longstring marker, %s", err)
	}
	n = n + 1

	// U32
	if err = binary.Write(w, binary.BigEndian, uint32(len(val))); err != nil {
		return n, fmt.Errorf("amf0: error to write longstring length, %s", err)
	}
	n = n + 4

	// *(UTF8-char)
	if _, err = w.Write([]byte(val)); err != nil {
		return n, fmt.Errorf("amf0: error to write longstring value, %s", err)
	}
	return n + len(val), nil
}

// WriteUnsupported .
//  - unsupported-marker
func (amf *AMF0) WriteUnsupported(w Writer) (n int, err error) {
	if err = w.WriteByte(AMF0Unsupported); err != nil {
		return n, fmt.Errorf("amf0: error to write unsupported marker, %s", err)
	}
	return 1, nil
}
