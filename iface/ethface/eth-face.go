package ethface

/*
#include "../../csrc/ethface/eth-face.h"
*/
import "C"
import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/iface"
)

// New creates an Ethernet face on the given port.
func New(port *Port, loc Locator) (iface.Face, error) {
	if !loc.Local.IsZero() && !loc.Local.Equal(port.cfg.Local) {
		return nil, errors.New("port has a different local address")
	}
	loc.Local = port.cfg.Local

	switch {
	case loc.Remote.IsZero():
		loc.Remote = NdnMcastAddr
		fallthrough
	case loc.Remote.IsGroup():
		if face := port.FindFace(nil); face != nil {
			return nil, fmt.Errorf("port has another face %d with a group address", face.ID())
		}
	case loc.Remote.IsUnicast():
		if face := port.FindFace(&loc.Remote); face != nil {
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

			vlan := make([]uint16, 2)
			copy(vlan, loc.Vlan)
			priv.txHdrLen = C.EthFaceEtherHdr_Init(&priv.txHdr,
				(*C.struct_rte_ether_addr)(port.cfg.Local.Ptr()),
				(*C.struct_rte_ether_addr)(face.loc.Remote.Ptr()),
				C.uint16_t(vlan[0]), C.uint16_t(vlan[1]))

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
