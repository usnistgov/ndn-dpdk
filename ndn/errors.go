package ndn

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// Simple error conditions.
var (
	ErrFragment      = errors.New("bad fragment")
	ErrL3Type        = errors.New("unknown L3 packet type")
	ErrComponentType = errors.New("NameComponent TLV-TYPE out of range")
	ErrNonceLen      = errors.New("Nonce wrong length")
	ErrLifetime      = errors.New("InterestLifetime out of range")
	ErrHopLimit      = errors.New("HopLimit out of range")
	ErrParamsDigest  = errors.New("bad ParamsDigest")
	ErrSigType       = errors.New("bad SigType")
	ErrKeyLocator    = errors.New("bad KeyLocator")
	ErrSigNonce      = errors.New("bad SigNonce")
	ErrSigValue      = errors.New("bad SigValue")
)

func unmarshalNNI(de tlv.DecodingElement, max uint64, err *error, rangeErr error) (v uint64) {
	var n tlv.NNI
	if e := de.UnmarshalValue(&n); e != nil {
		*err = e
		return 0
	}

	v = uint64(n)
	if v > max {
		*err = rangeErr
		return 0
	}

	*err = nil
	return v
}
