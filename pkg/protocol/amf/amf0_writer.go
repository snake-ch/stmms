package amf

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"
)

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
	if err = w.WriteByte(amf0Number); err != nil {
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
	if err = w.WriteByte(amf0Boolean); err != nil {
		return n, fmt.Errorf("amf0: error to write boolean marker, %s", err)
	}
	n = n + 1

	// U8 (0 is false, <> 0 is true)
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
	if err = w.WriteByte(amf0String); err != nil {
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
	if err = w.WriteByte(amf0Object); err != nil {
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

// WriteMovieclip not supported and is reserved for future use.
func (amf *AMF0) WriteMovieclip(w Writer, val interface{}) (n int, err error) {
	panic(fmt.Errorf("amf0: not support write movieclip value, %s", err))
}

// WriteNull .
// 	- null-marker
func (amf *AMF0) WriteNull(w Writer) (n int, err error) {
	if err = w.WriteByte(amf0Null); err != nil {
		return n, fmt.Errorf("amf0: error to write null marker, %s", err)
	}
	return 1, nil
}

// WriteUndefined .
// 	- undefined-marker
func (amf *AMF0) WriteUndefined(w Writer) (n int, err error) {
	if err = w.WriteByte(amf0Undefined); err != nil {
		return n, fmt.Errorf("amf0: error to write undefined marker, %s", err)
	}
	return 1, nil
}

// WriteReference .
//  - reference-marker U16
func (amf *AMF0) WriteReference(w Writer, val uint16) (n int, err error) {
	panic(fmt.Errorf("amf0: not support to write reference value, %s", err))
}

// WriteEcmaArray .
//  - associative-count *((UTF-8 value-type) | (UTF-8-empty object-end-marker))
func (amf *AMF0) WriteEcmaArray(w Writer, val map[string]interface{}) (n int, err error) {
	// ecma-array marker
	if err = w.WriteByte(amf0EcmaArray); err != nil {
		return n, fmt.Errorf("amf0: error to write ecamarray marker, %s", err)
	}
	n = n + 1

	// associative-count
	if err = binary.Write(w, binary.BigEndian, uint32(len(val))); err != nil {
		return n, fmt.Errorf("amf0: error to write ecamarray length, %s", err)
	}
	n = n + 4

	number := 0
	for key, value := range val {
		// U16
		if err = binary.Write(w, binary.BigEndian, uint16(len(key))); err != nil {
			return n, fmt.Errorf("amf0: error to write index of ecamarray element, %s", err)
		}
		n = n + 2

		// *(UTF8-char)
		if number, err = w.Write([]byte(key)); err != nil {
			return n, fmt.Errorf("amf0: error to write key of ecamarray element, %s", err)
		}
		n = n + number

		// value-type
		number, err = amf.WriteTo(w, value)
		if err != nil {
			return n, fmt.Errorf("amf0: error to write value of ecamarray element, %s", err)
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
	return w.Write([]byte{0x00, 0x00, amf0ObjectEnd})
}

// WriteStrictArray .
//  - array-count *(value-type)
func (amf *AMF0) WriteStrictArray(w Writer, val []interface{}) (n int, err error) {
	// strict-array marker
	if err = w.WriteByte(amf0StrictArray); err != nil {
		return n, fmt.Errorf("amf0: error to write strictarray marker, %s", err)
	}
	n = n + 1

	// array-count
	if err = binary.Write(w, binary.BigEndian, uint32(len(val))); err != nil {
		return n, fmt.Errorf("amf0: error to write strictarray length, %s", err)
	}
	n = n + 4

	// *(value-type)
	number := 0
	for _, value := range val {
		number, err = amf.WriteTo(w, value)
		if err != nil {
			return n, fmt.Errorf("amf0: error to write strictarray element, %s", err)
		}
		n = n + number
	}
	return
}

// WriteDate .
//  - date-marker DOUBLE time-zone
func (amf *AMF0) WriteDate(w Writer, val time.Time) (n int, err error) {
	panic(fmt.Errorf("amf0: not support to write date value, %s", err))
}

// WriteLongString .
//  - long-string-marker UTF-8-long
func (amf *AMF0) WriteLongString(w Writer, val string) (n int, err error) {
	// long-string-marker
	if err = w.WriteByte(amf0LongString); err != nil {
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
	if err = w.WriteByte(amf0Unsupported); err != nil {
		return n, fmt.Errorf("amf0: error to write unsupported marker, %s", err)
	}
	return 1, nil
}

// WriteRecordSet not supported and is reserved for future use.
func (amf *AMF0) WriteRecordSet(w Writer, val interface{}) (n int, err error) {
	panic(fmt.Errorf("amf0: not support to write recordset value, %s", err))
}

// WriteXMLDocument .
//  - xml-document-marker UTF-8-long
func (amf *AMF0) WriteXMLDocument(w Writer, val string) (n int, err error) {
	panic(fmt.Errorf("amf0: not support to write xml document value, %s", err))
}

// WriteAcmplusObject .
//  - object-marker class-name *(object-property)
func (amf *AMF0) WriteAcmplusObject(w Writer) (n int, err error) {
	if err = w.WriteByte(amf0AvmplusObject); err != nil {
		return n, fmt.Errorf("amf0: error to write avmplus object marker, %s", err)
	}
	return 1, nil
}
