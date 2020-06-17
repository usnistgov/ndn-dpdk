package tlv

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

// Unmarshal unmarshals into a value.
func (d *Decoder) Unmarshal(u Unmarshaler) error {
	rest, e := u.UnmarshalTlv([]byte(*d))
	if e != nil {
		return e
	}
	*d = rest
	return nil
}

// DecodeFirst extracts the first TLV element.
func DecodeFirst(wire []byte) (element Element, rest []byte, e error) {
	rest, e = element.UnmarshalTlv(wire)
	return
}

// DecodeFirstExpect extracts the first TLV element, expecting a specified TLV-TYPE.
func DecodeFirstExpect(typ interface{}, wire []byte) (element Element, rest []byte, e error) {
	rest, e = element.UnmarshalTlv(wire)
	expectType := uint32(toUint((typ)))
	if element.Type != expectType {
		return Element{}, nil, ErrTypeExpect(expectType)
	}
	return
}
