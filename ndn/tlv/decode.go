package tlv

import (
	"encoding"
)

// Unmarshaler is the interface implemented by an object that can decode an TLV element representation of itself.
type Unmarshaler interface {
	UnmarshalTLV(typ uint32, value []byte) error
}

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

// Element recognizes one TLV element from the buffer.
func (d *DecodingBuffer) Element() (de DecodingElement, e error) {
	var typ, length VarNum

	afterTyp, e := typ.Decode(*d)
	if e != nil {
		return de, e
	}
	if typ < minType || typ > maxType {
		return de, ErrType
	}

	afterLen, e := length.Decode(afterTyp)
	if e != nil {
		return de, e
	}
	if len(afterLen) < int(length) {
		return de, ErrIncomplete
	}

	de.Type = uint32(typ)
	de.Value = afterLen[:length]
	de.After = afterLen[length:]
	de.Wire = (*d)[:len(*d)-len(de.After)]
	*d = de.After
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
	return de.Type <= 31 || de.Type&0x01 != 0
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

// UnmarshalNNI unmarshals TLV-VALUE as NNI and checks it's within [0:max] range.
//
// This function has an unusual set of parameters to allow for more compact calling code:
//
//	if pkt.Field = uint32(de.UnmarshalNNI(math.MaxUint32, &e, tlv.ErrRange)); e != nil {
//	  return e
//	}
func (de DecodingElement) UnmarshalNNI(max uint64, err *error, rangeErr error) (v uint64) {
	var n NNI
	if e := de.UnmarshalValue(&n); e != nil {
		*err = e
		return 0
	}

	v = uint64(n)
	if v > max {
		*err = rangeErr
		return 0
	}

	*err = nil
	return v
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
