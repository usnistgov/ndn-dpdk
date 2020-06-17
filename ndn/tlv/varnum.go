package tlv

import (
	"math"
	"reflect"
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
	case n <= math.MaxInt32:
		return 5
	default:
		return 9
	}
}

// MarshalTlv encodes this number.
func (n VarNum) MarshalTlv() (wire []byte, e error) {
	switch {
	case n < 0xFD:
		return []byte{byte(n)}, nil
	case n <= math.MaxUint16:
		return []byte{0xFD, byte(n >> 8), byte(n)}, nil
	case n <= math.MaxInt32:
		return []byte{0xFE, byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}, nil
	default:
		return []byte{0xFF, byte(n >> 56), byte(n >> 48), byte(n >> 40), byte(n >> 32),
			byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}, nil
	}
}

// UnmarshalTlv decodes this number.
func (n *VarNum) UnmarshalTlv(wire []byte) (rest []byte, e error) {
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

func toUint(input interface{}) uint64 {
	val := reflect.ValueOf(input)
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := val.Int()
		if n < 0 {
			panic("uint cannot be negative")
		}
		return uint64(n)
	default:
		return val.Uint()
	}
}
