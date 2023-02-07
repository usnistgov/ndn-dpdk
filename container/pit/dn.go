package pit

/*
#include "../../csrc/pcct/pit-dn.h"
*/
import "C"
import (
	"bytes"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// DnRecord represents a PIT downstream record.
type DnRecord struct {
	c     *C.PitDn
	entry *Entry
}

// FaceID returns the face ID.
func (dn DnRecord) FaceID() iface.ID {
	return iface.ID(dn.c.face)
}

// PitToken returns the last received PIT token.
func (dn DnRecord) PitToken() (token []byte) {
	tokenC := &dn.c.token
	return bytes.Clone(cptr.AsByteSlice(tokenC.value[:tokenC.length]))
}

// Nonce returns the last received Nonce.
func (dn DnRecord) Nonce() ndn.Nonce {
	return ndn.NonceFromUint(uint32(dn.c.nonce))
}

// Expiry returns a timestamp when this record expires.
func (dn DnRecord) Expiry() eal.TscTime {
	return eal.TscTime(dn.c.expiry)
}
