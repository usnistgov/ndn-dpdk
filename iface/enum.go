package iface

//go:generate go run ../mk/enumgen/ -guard=NDNDPDK_IFACE_ENUM_H -out=../csrc/iface/enum.h .

import (
	"strconv"
)

const (
	// MaxBurstSize is the maximum and default burst size.
	MaxBurstSize = 64

	// MaxInputDemuxDest is the maximum number of destinations in InputDemux.
	MaxInputDemuxDest = 128

	// MaxRxProcThreads is the maximum number of RX threads in RxProc.
	MaxRxProcThreads = 8

	// MinReassemblerCapacity is the minimum partial message store capacity in the reassembler.
	MinReassemblerCapacity = 4

	// MaxReassemblerCapacity is the maximum partial message store capacity in the reassembler.
	MaxReassemblerCapacity = 8192

	// DefaultReassemblerCapacity is the default partial message store capacity in the reassembler.
	DefaultReassemblerCapacity = 64

	// MinOutputQueueSize is the minimum packet queue capacity before the output thread.
	MinOutputQueueSize = 256

	// DefaultOutputQueueSize is the default packet queue capacity before the output thread.
	DefaultOutputQueueSize = 1024

	// MinMTU is the minimum value of Maximum Transmission Unit (MTU).
	MinMTU = 960

	// MaxMTU is the maximum value of Maximum Transmission Unit (MTU).
	MaxMTU = 65000

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

	_ = "enumgen:FaceState:Face"
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
