// Package tlv implements Type-Length-Value (TLV) encoding in Named Data Networking (NDN).
package tlv

// Marshaler is the interface implemented by an object that can encode itself into an TLV element.
type Marshaler interface {
	MarshalTlv() (typ uint32, value []byte, e error)
}

// Unmarshaler is the interface implemented by an object that can decode an TLV element representation of itself.
type Unmarshaler interface {
	UnmarshalTlv(typ uint32, value []byte) error
}
