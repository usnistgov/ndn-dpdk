package tlv

import "errors"

// Error conditions.
var (
	ErrIncomplete = errors.New("incomplete input")
	ErrTail       = errors.New("junk after end of TLV")
	ErrType       = errors.New("TLV-TYPE out of range")
	ErrCritical   = errors.New("unrecognized critical TLV-TYPE")
	ErrRange      = errors.New("out of range")
	ErrErrorField = errors.New("Error(nil) field")
)
