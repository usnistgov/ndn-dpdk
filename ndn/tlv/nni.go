package tlv

import (
	"encoding"
	"math"
)

// NNI is a non-negative integer.
type NNI uint64

var (
	_ Fielder                    = NNI(0)
	_ encoding.BinaryUnmarshaler = (*NNI)(nil)
)

// Size returns the wire encoding size.
func (n NNI) Size() int {
	switch {
	case n <= math.MaxUint8:
		return 1
	case n <= math.MaxUint16:
		return 2
	case n <= math.MaxUint32:
		return 4
	default:
		return 8
	}
}

// Encode encodes this number.
func (n NNI) Encode(b []byte) []byte {
	switch {
	case n <= math.MaxUint8:
		b = append(b, byte(n))
	case n <= math.MaxUint16:
		b = append(b, byte(n>>8), byte(n))
	case n <= math.MaxUint32:
		b = append(b, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	default:
		b = append(b, byte(n>>56), byte(n>>48), byte(n>>40), byte(n>>32),
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
	return b
}

// Field implements Fielder interface.
func (n NNI) Field() Field {
	return Field{
		typ:     fieldTypeNNI,
		integer: uint64(n),
	}
}

// UnmarshalBinary decodes this number.
func (n *NNI) UnmarshalBinary(wire []byte) error {
	switch len(wire) {
	case 1:
		*n = NNI(wire[0])
	case 2:
		*n = (NNI(wire[0]) << 8) | NNI(wire[1])
	case 4:
		*n = (NNI(wire[0]) << 24) | (NNI(wire[1]) << 16) | (NNI(wire[2]) << 8) | NNI(wire[3])
	case 8:
		*n = (NNI(wire[0]) << 56) | (NNI(wire[1]) << 48) | (NNI(wire[2]) << 40) | (NNI(wire[3]) << 32) |
			(NNI(wire[4]) << 24) | (NNI(wire[5]) << 16) | (NNI(wire[6]) << 8) | NNI(wire[7])
	default:
		return ErrIncomplete
	}
	return nil
}
