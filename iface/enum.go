package iface

//go:generate go run ../mk/enumgen/ -guard=NDNDPDK_IFACE_ENUM_H -out=../csrc/iface/enum.h .

import (
	"strconv"
)

const (
	// MaxBurstSize is the maximum and default burst size.
	MaxBurstSize = 64

	// MinMtu is the minimum value of Maximum Transmission Unit (MTU).
	MinMtu = 1280

	// MaxMtu is the maximum value of Maximum Transmission Unit (MTU).
	MaxMtu = 65000

	// MinReassemblerCapacity is the minimum partial message store capacity in the reassembler.
	MinReassemblerCapacity = 4

	// MaxReassemblerCapacity is the maximum partial message store capacity in the reassembler.
	MaxReassemblerCapacity = 8192

	// DefaultReassemblerCapacity is the default partial message store capacity in the reassembler.
	DefaultReassemblerCapacity = 64

	_ = "enumgen"
)

// State indicates face state.
type State uint8

// State values.
const (
	StateUnused = iota
	StateUp
	StateDown
	StateRemoved

	_ = "enumgen:FaceState-State"
)

func (st State) String() string {
	switch st {
	case StateUnused:
		return "unused"
	case StateUp:
		return "up"
	case StateDown:
		return "down"
	case StateRemoved:
		return "removed"
	}
	return strconv.Itoa(int(st))
}
