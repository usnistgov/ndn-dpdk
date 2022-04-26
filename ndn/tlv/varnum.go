// Package tlv implements NDN Type-Length-Value (TLV) encoding.
package tlv

import "math"

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
		return append(buf, 0xFD, byte(n>>8), byte(n))
	case n <= math.MaxUint32:
		return append(buf, 0xFE, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	default:
		return append(buf, 0xFF, byte(n>>56), byte(n>>48), byte(n>>40), byte(n>>32),
			byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
	}
}

// Decode extracts a VarNum from the buffer.
func (n *VarNum) Decode(wire []byte) (rest []byte, e error) {
	switch {
	case len(wire) >= 1 && wire[0] < 0xFD:
		*n = VarNum(wire[0])
		return wire[1:], nil
	case len(wire) >= 3 && wire[0] == 0xFD:
		*n = (VarNum(wire[1]) << 8) | VarNum(wire[2])
		return wire[3:], nil
	case len(wire) >= 5 && wire[0] == 0xFE:
		*n = (VarNum(wire[1]) << 24) | (VarNum(wire[2]) << 16) | (VarNum(wire[3]) << 8) | VarNum(wire[4])
		return wire[5:], nil
	case len(wire) >= 9 && wire[0] == 0xFF:
		*n = (VarNum(wire[1]) << 56) | (VarNum(wire[2]) << 48) | (VarNum(wire[3]) << 40) | (VarNum(wire[4]) << 32) |
			(VarNum(wire[5]) << 24) | (VarNum(wire[6]) << 16) | (VarNum(wire[7]) << 8) | VarNum(wire[8])
		return wire[9:], nil
	}
	return nil, ErrIncomplete
}
