// Package fibdef declares common data structures for FIB.
package fibdef

import "errors"

//go:generate go run ../../../mk/enumgen/ -guard=NDNDPDK_FIB_ENUM_H -out=../../../csrc/fib/enum.h .

const (
	// MaxNameLength is the maximum TLV-LENGTH of a FIB entry name.
	MaxNameLength = 494

	// MaxNexthops is the maximum number of nexthops in a FIB entry.
	MaxNexthops = 8

	// ScratchSize is the size of strategy scratch area.
	ScratchSize = 96

	_ = "enumgen::Fib"
)

// Errors.
var (
	ErrNameTooLong = errors.New("FIB entry name too long")
	ErrNexthops    = errors.New("number of nexthops out of range")
	ErrStrategy    = errors.New("missing strategy")
)
