package tlv

import "math"

// Element represents a TLV element.
// The zero Element is invalid.
type Element struct {
	// Type is the TLV-TYPE.
	Type uint32
	// Value is the TLV-VALUE.
	Value []byte
}

// MakeElement constructs Element from TLV-TYPE and TLV-VALUE.
// typ can be any integer type.
func MakeElement(typ uint32, value []byte) (element Element) {
	element.Type = typ
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

// Decode extracts an element from the buffer.
func (element *Element) Decode(wire []byte) (rest []byte, e error) {
	var typ, length VarNum
	if wire, e = typ.Decode(wire); e != nil {
		return nil, e
	}
	if typ < minType || typ > maxType {
		return nil, ErrType
	}
	if wire, e = length.Decode(wire); e != nil {
		return nil, e
	}
	if len(wire) < int(length) {
		return nil, ErrIncomplete
	}
	element.UnmarshalTlv(uint32(typ), wire[:length])
	return wire[length:], nil
}

// MarshalTlv implements Marshaler interface.
func (element Element) MarshalTlv() (typ uint32, value []byte, e error) {
	return element.Type, element.Value, nil
}

// UnmarshalTlv implements Unmarshaler interface.
func (element *Element) UnmarshalTlv(typ uint32, value []byte) error {
	element.Type = typ
	element.Value = value
	return nil
}

const (
	minType = 1
	maxType = math.MaxUint32
)
