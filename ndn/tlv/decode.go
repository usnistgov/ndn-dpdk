package tlv

import (
	"encoding"
	"reflect"
)

// Decoder recognizes TLV elements.
type Decoder []byte

// Rest returns unconsumed input.
func (d Decoder) Rest() []byte {
	return []byte(d)
}

// EOF returns true if decoder is at end of input.
func (d Decoder) EOF() bool {
	return len(d) == 0
}

// ErrUnlessEOF returns an error if there is unconsumed input.
func (d Decoder) ErrUnlessEOF() error {
	if d.EOF() {
		return nil
	}
	return ErrTail
}

// Element recognizes one TLV element from start of input.
func (d *Decoder) Element() (de DecoderElement, e error) {
	rest, e := de.Decode(*d)
	if e != nil {
		return DecoderElement{}, e
	}
	de.Wire = (*d)[:len(*d)-len(rest)]
	de.After = rest
	*d = rest
	return de, nil
}

// Elements recognizes TLV elements from start of input.
// Bytes that cannot be recognized as TLV elements are left in the decoder.
func (d *Decoder) Elements() (list []DecoderElement) {
	for {
		de, e := d.Element()
		if e != nil {
			break
		}
		list = append(list, de)
	}
	return list
}

// DecoderElement represents an TLV element during decoding.
type DecoderElement struct {
	Element
	Wire  []byte
	After []byte
}

// IsCriticalType returns true if the TLV-TYPE is considered critical for evolvability purpose.
func (de DecoderElement) IsCriticalType() bool {
	return de.Type <= 31 || (de.Type&0x01) != 0
}

// WireAfter returns Wire+After.
func (de DecoderElement) WireAfter() []byte {
	size := len(de.Wire) + len(de.After)
	if cap(de.Wire) < size {
		panic(de.Wire)
	}
	return de.Wire[:size]
}

// Unmarshal unmarshals TLV into a value.
func (de DecoderElement) Unmarshal(u Unmarshaler) error {
	return u.UnmarshalTlv(de.Type, de.Value)
}

// UnmarshalValue unmarshals TLV-VALUE into a value.
func (de DecoderElement) UnmarshalValue(u encoding.BinaryUnmarshaler) error {
	return u.UnmarshalBinary(de.Value)
}

// UnmarshalNNI unmarshals TLV-VALUE as NNI, written to a pointer of integer type.
func (de DecoderElement) UnmarshalNNI(ptr interface{}) error {
	var n NNI
	if e := de.UnmarshalValue(&n); e != nil {
		return e
	}

	val := reflect.ValueOf(ptr).Elem()
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val.SetInt(int64(n))
	default:
		val.SetUint(uint64(n))
	}
	return nil
}

// Decode unmarshals a buffer that contains a single TLV.
func Decode(wire []byte, u Unmarshaler) error {
	d := Decoder(wire)
	de, e := d.Element()
	if e != nil {
		return e
	}
	if e = de.Unmarshal(u); e != nil {
		return e
	}
	return d.ErrUnlessEOF()
}
