package tlv

import "math"

// Element represents a TLV element.
// Zero value is invalid.
type Element struct {
	// Type is the TLV-TYPE.
	Type uint32
	// Value is the TLV-VALUE.
	Value []byte
}

var (
	_ Fielder     = Element{}
	_ Unmarshaler = &Element{}
)

// Size returns encoded size.
func (element Element) Size() int {
	length := element.Length()
	return VarNum(element.Type).Size() + VarNum(length).Size() + length
}

// Length returns TLV-LENGTH.
func (element Element) Length() int {
	return len(element.Value)
}

// Field implements Fielder interface.
func (element Element) Field() Field {
	return TLVBytes(element.Type, element.Value)
}

// UnmarshalTLV implements Unmarshaler interface.
func (element *Element) UnmarshalTLV(typ uint32, value []byte) error {
	element.Type = typ
	element.Value = value
	return nil
}

const (
	minType = 1
	maxType = math.MaxUint32
)
