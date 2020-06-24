package tlv

import (
	"bytes"
	"encoding"
	"reflect"
)

type encoderParts [][]byte

func (parts *encoderParts) Append(value interface{}) error {
	if b, ok := value.([]byte); ok {
		*parts = append(*parts, b)
		return nil
	}

	if m, ok := value.(Marshaler); ok {
		typ, val, e := m.MarshalTlv()
		if e != nil {
			return e
		}
		tl := VarNum(typ).Encode(nil)
		tl = VarNum(len(val)).Encode(tl)
		*parts = append(*parts, tl, val)
		return nil
	}

	if m, ok := value.(encoding.BinaryMarshaler); ok {
		b, e := m.MarshalBinary()
		if e != nil {
			return e
		}
		*parts = append(*parts, b)
		return nil
	}

	slice := reflect.ValueOf(value)
	count := slice.Len()
	for i := 0; i < count; i++ {
		if e := parts.Append(slice.Index(i).Interface()); e != nil {
			return e
		}
	}
	return nil
}

// Encode encodes a sequence of values.
// Each value can be []byte, Marshaler, encoding.BinaryMarshaler, or slice of them.
func Encode(values ...interface{}) (wire []byte, e error) {
	var parts encoderParts
	e = parts.Append(values)
	if e != nil {
		return nil, e
	}
	return bytes.Join([][]byte(parts), nil), nil
}

// EncodeTlv encodes a sequence of values into []byte.
// It can be used to implement Marshaler.
func EncodeTlv(typ uint32, values ...interface{}) (typ1 uint32, value []byte, e error) {
	value, e = Encode(values...)
	return typ, value, e
}
