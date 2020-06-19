package tlv

import (
	"errors"
)

// Simple error conditions.
var (
	ErrIncomplete = errors.New("incomplete input")
	ErrTail       = errors.New("junk after end of TLV")
	ErrType       = errors.New("TLV-TYPE out of range")
	ErrCritical   = errors.New("unrecognized critical TLV-TYPE")
)
