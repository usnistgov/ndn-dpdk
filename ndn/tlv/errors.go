package tlv

import (
	"errors"
	"strconv"
)

// Simple error conditions.
var (
	ErrIncomplete = errors.New("incomplete input")
	ErrTypeZero   = errors.New("TLV-TYPE cannot be zero")
	ErrTail       = errors.New("junk after end of TLV")
)

// ErrTypeExpect indicates that the input TLV-TYPE differs from an expected TLV-TYPE.
type ErrTypeExpect uint32

func (e ErrTypeExpect) Error() string {
	return "TLV-TYPE should be " + strconv.Itoa(int(e))
}
