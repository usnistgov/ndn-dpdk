package pit

/*
#include "../../csrc/pcct/pit-dn.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Dn represents a PIT downstream record.
type Dn struct {
	c     *C.PitDn
	entry *Entry
}

// FaceID returns the face ID.
func (dn Dn) FaceID() iface.ID {
	return iface.ID(dn.c.face)
}

// GetToken returns the last received PIT token.
func (dn Dn) GetToken() uint64 {
	return uint64(dn.c.token)
}

// GetNonce returns the last received Nonce.
func (dn Dn) GetNonce() ndn.Nonce {
	return ndn.NonceFromUint(uint32(dn.c.nonce))
}

// GetExpiry returns a timestamp when this record expires.
func (dn Dn) GetExpiry() eal.TscTime {
	return eal.TscTime(dn.c.expiry)
}
