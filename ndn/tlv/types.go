// Package tlv implements NDN Type-Length-Value (TLV) encoding.
package tlv

// Encoder is the interface implemented by an object that can encode itself to bytes.
type Encoder interface {
	// Encode encodes the object by appending to the given byte slice.
	Encode(b []byte) ([]byte, error)
}

// Fielder is the interface implemented by an object that can encode itself to a Field.
type Fielder interface {
	Field() Field
}

// Decoder is the interface implemented by an object that can decode itself from bytes.
type Decoder interface {
	// Decode decodes to the object and returns rest bytes.
	Decode(b []byte) (rest []byte, e error)
}

// Unmarshaler is the interface implemented by an object that can decode an TLV element representation of itself.
type Unmarshaler interface {
	UnmarshalTLV(typ uint32, value []byte) error
}
