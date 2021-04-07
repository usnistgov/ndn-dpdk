package tlv

import (
	"encoding"
)

// DecodingBuffer recognizes TLV elements.
type DecodingBuffer []byte

// Rest returns unconsumed input.
func (d DecodingBuffer) Rest() []byte {
	return []byte(d)
}

// EOF returns true if decoder is at end of input.
func (d DecodingBuffer) EOF() bool {
	return len(d) == 0
}

// ErrUnlessEOF returns an error if there is unconsumed input.
func (d DecodingBuffer) ErrUnlessEOF() error {
	if d.EOF() {
		return nil
	}
	return ErrTail
}

// Decode extracts one item from the buffer.
func (d *DecodingBuffer) Decode(item Decoder) error {
	rest, e := item.Decode(*d)
	if e != nil {
		return e
	}
	*d = rest
	return nil
}

// Element recognizes one TLV element from the buffer.
func (d *DecodingBuffer) Element() (de DecodingElement, e error) {
	wireAfter := *d
	if e = d.Decode(&de); e != nil {
		return DecodingElement{}, e
	}

	de.Wire = wireAfter[:len(wireAfter)-len(*d)]
	de.After = *d
	return de, nil
}

// Elements recognizes TLV elements from the buffer.
// Bytes that cannot be recognized as TLV elements are left in the decoder.
func (d *DecodingBuffer) Elements() (list []DecodingElement) {
	for {
		de, e := d.Element()
		if e != nil {
			break
		}
		list = append(list, de)
	}
	return list
}

// DecodingElement represents an TLV element during decoding.
type DecodingElement struct {
	Element
	Wire  []byte
	After []byte
}

// IsCriticalType returns true if the TLV-TYPE is considered critical for evolvability purpose.
func (de DecodingElement) IsCriticalType() bool {
	return de.Type <= 31 || (de.Type&0x01) != 0
}

// WireAfter returns Wire+After.
func (de DecodingElement) WireAfter() []byte {
	size := len(de.Wire) + len(de.After)
	if cap(de.Wire) < size {
		panic(de.Wire)
	}
	return de.Wire[:size]
}

// Unmarshal unmarshals TLV into a value.
func (de DecodingElement) Unmarshal(u Unmarshaler) error {
	return u.UnmarshalTLV(de.Type, de.Value)
}

// UnmarshalValue unmarshals TLV-VALUE into a value.
func (de DecodingElement) UnmarshalValue(u encoding.BinaryUnmarshaler) error {
	return u.UnmarshalBinary(de.Value)
}

// Decode unmarshals a buffer that contains a single TLV.
func Decode(wire []byte, u Unmarshaler) error {
	d := DecodingBuffer(wire)
	de, e := d.Element()
	if e != nil {
		return e
	}
	if e = de.Unmarshal(u); e != nil {
		return e
	}
	return d.ErrUnlessEOF()
}
