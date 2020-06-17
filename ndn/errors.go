package ndn

import (
	"errors"
)

// Simple error conditions.
var (
	ErrComponentType = errors.New("NameComponent TLV-TYPE out of range")
)
