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
	d := DecodingBuffer(wire)
	var typ, length VarNum

	if e = d.Decode(&typ); e != nil {
		return nil, e
	}
	if typ < minType || typ > maxType {
		return nil, ErrType
	}

	if e = d.Decode(&length); e != nil {
		return nil, e
	}

	valueRest := d.Rest()
	if len(valueRest) < int(length) {
		return nil, ErrIncomplete
	}
	element.Type = uint32(typ)
	element.Value = valueRest[:length]
	return valueRest[length:], nil
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
