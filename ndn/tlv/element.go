package tlv

import (
	"bytes"
)

// Element represents a TLV element.
// The zero Element is invalid.
type Element struct {
	// Type is the TLV-TYPE.
	Type uint32
	// Value is the TLV-VALUE.
	Value []byte
}

// MakeElement constructs Element from TLV-TYPE and TLV-VALUE.
// typ can be any integral type.
func MakeElement(typ interface{}, value []byte) (element Element) {
	element.Type = uint32(toUint(typ))
	element.Value = value
	return element
}

// Size returns encoded size.
func (element Element) Size() int {
	return VarNum(element.Type).Size() + VarNum(element.Length()).Size() + len(element.Value)
}

// Length returns TLV-LENGTH.
func (element Element) Length() int {
	return len(element.Value)
}

// MarshalTlv encodes this element.
func (element Element) MarshalTlv() (wire []byte, e error) {
	if element.Type == 0 {
		return nil, ErrTypeZero
	}
	typ, _ := VarNum(element.Type).MarshalTlv()
	length, _ := VarNum(element.Length()).MarshalTlv()
	return bytes.Join([][]byte{typ, length, element.Value}, nil), nil
}

// UnmarshalTlv decodes from wire format.
func (element *Element) UnmarshalTlv(wire []byte) (rest []byte, e error) {
	var typ, length VarNum
	if wire, e = typ.UnmarshalTlv(wire); e != nil {
		return nil, e
	}
	if wire, e = length.UnmarshalTlv(wire); e != nil {
		return nil, e
	}
	if len(wire) < int(length) {
		return nil, ErrIncomplete
	}
	element.Type = uint32(typ)
	element.Value = wire[:length]
	return wire[length:], nil
}
