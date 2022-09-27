// Package tlv implements NDN Type-Length-Value (TLV) encoding.
package tlv

import (
	"encoding/binary"
	"math"
)

// VarNum represents a number in variable size encoding for TLV-TYPE or TLV-LENGTH.
type VarNum uint64

// Size returns the wire encoding size.
func (n VarNum) Size() int {
	switch {
	case n < 0xFD:
		return 1
	case n <= math.MaxUint16:
		return 3
	case n <= math.MaxUint32:
		return 5
	default:
		return 9
	}
}

// Encode appends this number to a buffer.
func (n VarNum) Encode(buf []byte) []byte {
	switch {
	case n < 0xFD:
		return append(buf, byte(n))
	case n <= math.MaxUint16:
		return binary.BigEndian.AppendUint16(append(buf, 0xFD), uint16(n))
	case n <= math.MaxUint32:
		return binary.BigEndian.AppendUint32(append(buf, 0xFE), uint32(n))
	default:
		return binary.BigEndian.AppendUint64(append(buf, 0xFF), uint64(n))
	}
}

// Decode extracts a VarNum from the buffer.
func (n *VarNum) Decode(wire []byte) (rest []byte, e error) {
	switch {
	case len(wire) >= 1 && wire[0] < 0xFD:
		*n = VarNum(wire[0])
		return wire[1:], nil
	case len(wire) >= 3 && wire[0] == 0xFD:
		*n = VarNum(binary.BigEndian.Uint16(wire[1:]))
		return wire[3:], nil
	case len(wire) >= 5 && wire[0] == 0xFE:
		*n = VarNum(binary.BigEndian.Uint32(wire[1:]))
		return wire[5:], nil
	case len(wire) >= 9 && wire[0] == 0xFF:
		*n = VarNum(binary.BigEndian.Uint64(wire[1:]))
		return wire[9:], nil
	}
	return nil, ErrIncomplete
}
