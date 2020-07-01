package iface

//go:generate go run ../mk/enumgen/ -guard=NDN_DPDK_IFACE_ENUM_H -out=../csrc/iface/enum.h .

import (
	"strconv"
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
