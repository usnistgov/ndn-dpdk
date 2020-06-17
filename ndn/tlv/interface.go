package tlv

// Marshaler is the interface implemented by an object that can marshal itself into NDN-TLV.
type Marshaler interface {
	MarshalTlv() (wire []byte, e error)
}

// Unmarshaler is the interface implemented by an object that can marshal an NDN-TLV representation of itself.
type Unmarshaler interface {
	UnmarshalTlv(wire []byte) (rest []byte, e error)
}
