package tlv

import (
	"bytes"
	"reflect"
)

func encodeSequence(headroom []byte, values ...interface{}) (wire []byte, e error) {
	var marshalers []Marshaler
	for _, value := range values {
		if marshaler, ok := value.(Marshaler); ok {
			marshalers = append(marshalers, marshaler)
			continue
		}

		val := reflect.ValueOf(value)
		count := val.Len()
		for i := 0; i < count; i++ {
			marshalers = append(marshalers, val.Index(i).Interface().(Marshaler))
		}
	}

	parts := make([][]byte, 1+len(marshalers))
	parts[0] = headroom
	for i, marshaler := range marshalers {
		parts[i+1], e = marshaler.MarshalTlv()
		if e != nil {
			return nil, e
		}
	}
	return bytes.Join(parts, nil), nil
}

// EncodeValue encodes a sequence of TLV structures.
// Each of values is either a Marshaler or a slice of Marshalers.
func EncodeValue(values ...interface{}) (wire []byte, e error) {
	return encodeSequence(nil, values...)
}

// EncodeElement encodes a TLV structure.
// typ is a integral type for TLV-TYPE.
// Each of values is either a Marshaler or a slice of Marshalers.
func EncodeElement(typ interface{}, values ...interface{}) (wire []byte, e error) {
	wire, e = encodeSequence(marshalTypeLengthPlaceholder, values...)
	if e != nil {
		return nil, e
	}
	length := len(wire) - marshalTypeLengthPlaceholderSize

	typeWire, _ := VarNum(toUint(typ)).MarshalTlv()
	lengthWire, _ := VarNum(length).MarshalTlv()
	lengthOffset := marshalTypeLengthPlaceholderSize - len(lengthWire)
	typeOffset := lengthOffset - len(typeWire)
	copy(wire[typeOffset:], typeWire)
	copy(wire[lengthOffset:], lengthWire)
	return wire[typeOffset:], nil
}

const marshalTypeLengthPlaceholderSize = 14

var marshalTypeLengthPlaceholder = make([]byte, marshalTypeLengthPlaceholderSize)
