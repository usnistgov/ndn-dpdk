package ndn

import (
	"errors"
)

// Simple error conditions.
var (
	ErrL3Type        = errors.New("unknown L3 packet type")
	ErrComponentType = errors.New("NameComponent TLV-TYPE out of range")
	ErrNonceLen      = errors.New("Nonce wrong length")
	ErrLifetime      = errors.New("InterestLifetime out of range")
	ErrHopLimit      = errors.New("HopLimit out of range")
)
