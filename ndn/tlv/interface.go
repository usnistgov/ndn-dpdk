// Package tlv implements NDN Type-Length-Value (TLV) encoding.
package tlv

// Marshaler is the interface implemented by an object that can encode itself into an TLV element.
type Marshaler interface {
	MarshalTlv() (typ uint32, value []byte, e error)
}

// Unmarshaler is the interface implemented by an object that can decode an TLV element representation of itself.
type Unmarshaler interface {
	UnmarshalTlv(typ uint32, value []byte) error
}
