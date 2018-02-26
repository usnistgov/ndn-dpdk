package pit

/*
#include "../pcct/pit-dn.h"
*/
import "C"
import (
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
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

func (dn Dn) GetNonce() uint32 {
	return uint32(dn.c.nonce)
}

func (dn Dn) GetExpiry() dpdk.TscTime {
	return dpdk.TscTime(dn.c.expiry)
}
