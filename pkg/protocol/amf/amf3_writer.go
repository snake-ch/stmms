package amf

import (
	"encoding/binary"
	"fmt"
	"reflect"
	"time"
)

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
	if err = w.WriteByte(amf3Undefined); err != nil {
		return n, fmt.Errorf("amf3: error to write undefined marker, %s", err)
	}
	return n + 1, nil
}

// WriteNull .
//  - null-marker
func (amf *AMF3) WriteNull(w Writer) (n int, err error) {
	if err = w.WriteByte(amf3Null); err != nil {
		return n, fmt.Errorf("amf3: error to write null marker, %s", err)
	}
	return n + 1, nil
}

// WriteFalse .
//  - false-marker
func (amf *AMF3) WriteFalse(w Writer) (n int, err error) {
	if err = w.WriteByte(amf3False); err != nil {
		return n, fmt.Errorf("amf3: error to write false marker, %s", err)
	}
	return n + 1, nil
}

// WriteTrue .
//  - true-marker
func (amf *AMF3) WriteTrue(w Writer) (n int, err error) {
	if err = w.WriteByte(amf3True); err != nil {
		return n, fmt.Errorf("amf3: error to write true marker, %s", err)
	}
	return n + 1, nil
}

// WriteInteger .
//  - interger-marker U29
func (amf *AMF3) WriteInteger(w Writer, val uint32) (n int, err error) {
	// interger-marker
	if err = w.WriteByte(amf3Integer); err != nil {
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
	if err = w.WriteByte(amf3Double); err != nil {
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
	if err = w.WriteByte(amf3String); err != nil {
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

// WriteXMLDoc .
func (amf *AMF3) WriteXMLDoc(w Writer, val uint32) (n int, err error) {
	panic(fmt.Errorf("amf3: not support to write xml document value, %s", err))
}

// WriteDate .
//  - date-marker U29O-ref
//  ✔ date-marker U29D-value date-time
func (amf *AMF3) WriteDate(w Writer, val time.Time) (n int, err error) {
	// date-marker
	if err = w.WriteByte(amf3Date); err != nil {
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
	if err = w.WriteByte(amf3Array); err != nil {
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
	if err = w.WriteByte(amf3Object); err != nil {
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

// WriteXML .
func (amf *AMF3) WriteXML(w Writer, val interface{}) (n int, err error) {
	panic(fmt.Errorf("amf3: not support to write xml value, %s", err))
}

// WriteBytearray .
//  - bytearray-marker U29O-ref
//  ✔ bytearray-marker U29B-value *(U8)
func (amf *AMF3) WriteBytearray(w Writer, val []byte) (n int, err error) {
	// bytearray marker
	if err = w.WriteByte(amf3ByteArray); err != nil {
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

// WriteVectorInt .
func (amf *AMF3) WriteVectorInt(w Writer, val interface{}) (n int, err error) {
	panic(fmt.Errorf("amf3: not support to write Vector Int value, %s", err))
}

// WriteVectorUint .
func (amf *AMF3) WriteVectorUint(w Writer, val interface{}) (n int, err error) {
	panic(fmt.Errorf("amf3: not support to write Vector Uint value, %s", err))
}

// WriteVectorDouble .
func (amf *AMF3) WriteVectorDouble(w Writer, val interface{}) (n int, err error) {
	panic(fmt.Errorf("amf3: not support to write Vector Double value, %s", err))
}

// WriteVectorObject .
func (amf *AMF3) WriteVectorObject(w Writer, val interface{}) (n int, err error) {
	panic(fmt.Errorf("amf3: not support to write Vector Object value, %s", err))
}

// WriteDictionary .
func (amf *AMF3) WriteDictionary(w Writer, val interface{}) (n int, err error) {
	panic(fmt.Errorf("amf3: not support to write Dictionary value, %s", err))
}
