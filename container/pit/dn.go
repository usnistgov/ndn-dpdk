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

// A PIT downstream record.
type Dn struct {
	c     *C.PitDn
	entry Entry
}

func (dn Dn) GetFaceId() iface.FaceId {
	return iface.FaceId(dn.c.face)
}

func (dn Dn) GetToken() uint64 {
	return uint64(dn.c.token)
}

func (dn Dn) GetNonce() ndn.Nonce {
	return ndn.NonceFromUint(uint32(dn.c.nonce))
}

func (dn Dn) GetExpiry() eal.TscTime {
	return eal.TscTime(dn.c.expiry)
}
