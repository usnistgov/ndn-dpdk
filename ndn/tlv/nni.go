package tlv

import (
	"encoding"
	"encoding/binary"
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
		b = binary.BigEndian.AppendUint16(b, uint16(n))
	case n <= math.MaxUint32:
		b = binary.BigEndian.AppendUint32(b, uint32(n))
	default:
		b = binary.BigEndian.AppendUint64(b, uint64(n))
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
		*n = NNI(binary.BigEndian.Uint16(wire))
	case 4:
		*n = NNI(binary.BigEndian.Uint32(wire))
	case 8:
		*n = NNI(binary.BigEndian.Uint64(wire))
	default:
		return ErrIncomplete
	}
	return nil
}
