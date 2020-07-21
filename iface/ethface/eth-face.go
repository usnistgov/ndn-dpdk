package ethface

/*
#include "../../csrc/ethface/eth-face.h"
*/
import "C"
import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// New creates an Ethernet face on the given port.
func New(port *Port, loc Locator) (iface.Face, error) {
	if !macaddr.Equal(loc.Local, port.cfg.Local) {
		return nil, errors.New("port has a different local address")
	}

	switch {
	case macaddr.IsMulticast(loc.Remote):
		if face := port.FindFace(nil); face != nil {
			return nil, fmt.Errorf("port has another face %d with a group address", face.ID())
		}
	case macaddr.IsUnicast(loc.Remote):
		if face := port.FindFace(loc.Remote); face != nil {
			return nil, fmt.Errorf("port has another face %d with same unicast address", face.ID())
		}
	default:
		return nil, fmt.Errorf("invalid MAC address")
	}

	face := &ethFace{
		port: port,
		loc:  loc,
	}
	return iface.New(iface.NewOptions{
		Socket:          port.dev.NumaSocket(),
		SizeofPriv:      uintptr(C.sizeof_EthFacePriv),
		TxQueueCapacity: port.cfg.TxqPkts,
		TxMtu:           port.cfg.Mtu,
		TxHeadroom:      int(C.sizeof_struct_rte_ether_hdr),
		Init: func(f iface.Face) error {
			face.Face = f
			c := face.ptr()
			c.txBurstOp = (C.FaceImpl_TxBurst)(C.EthFace_TxBurst)

			priv := (*C.EthFacePriv)(C.Face_GetPriv(c))
			priv.port = C.uint16_t(port.dev.ID())
			priv.faceID = C.FaceID(f.ID())

			var local, remote C.struct_rte_ether_addr
			copy(cptr.AsByteSlice(&local.addr_bytes), []byte(port.cfg.Local))
			copy(cptr.AsByteSlice(&remote.addr_bytes), []byte(face.loc.Remote))
			priv.txHdrLen = C.EthFaceEtherHdr_Init(&priv.txHdr, &local, &remote, C.uint16_t(face.loc.VLAN))

			face.priv = priv
			return nil
		},
		Start: func(iface.Face) (iface.Face, error) {
			return face, port.startFace(face, false)
		},
		Locator: func(iface.Face) iface.Locator {
			return face.loc
		},
		Stop: func(iface.Face) error {
			return face.port.stopFace(face)
		},
		Close: func(iface.Face) error {
			if face.port.CountFaces() == 0 {
				face.port.Close()
			}
			return nil
		},
		ReadExCounters: func(iface.Face) interface{} {
			var cnt ExCounters
			cnt.RxQueue = int(face.priv.rxQueue)
			return cnt
		},
	})
}

type ethFace struct {
	iface.Face
	port *Port
	loc  Locator
	priv *C.EthFacePriv
	rxf  *rxFlow
}

func (face *ethFace) ptr() *C.Face {
	return (*C.Face)(face.Ptr())
}

// ExCounters contains extended counters for Ethernet faces.
// This only contains the RX queue number of the port, while the actual counters are on the port.
type ExCounters struct {
	RxQueue int
}
