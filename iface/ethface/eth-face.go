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

type EthFace struct {
	iface.FaceBase
	port *Port
	loc  Locator
	rxf  *rxFlow
}

func New(port *Port, loc Locator) (face *EthFace, e error) {
	if !loc.Local.IsZero() && !loc.Local.Equal(port.cfg.Local) {
		return nil, errors.New("port has a different local address")
	}
	loc.Local = port.cfg.Local

	switch {
	case loc.Remote.IsZero():
		loc.Remote = NdnMcastAddr
		fallthrough
	case loc.Remote.IsGroup():
		if face = port.FindFace(nil); face != nil {
			return nil, fmt.Errorf("port has another face %d with a group address", face.ID())
		}
	case loc.Remote.IsUnicast():
		if face = port.FindFace(&loc.Remote); face != nil {
			return nil, fmt.Errorf("port has another face %d with same unicast address", face.ID())
		}
	default:
		return nil, fmt.Errorf("invalid MAC address")
	}

	face = new(EthFace)
	if e := face.InitFaceBase(iface.AllocID(), int(C.sizeof_EthFacePriv), port.dev.NumaSocket()); e != nil {
		return nil, e
	}
	face.port = port
	face.loc = loc

	priv := face.getPriv()
	priv.port = C.uint16_t(face.port.dev.ID())
	priv.faceID = C.FaceID(face.ID())

	vlan := make([]uint16, 2)
	copy(vlan, loc.Vlan)
	priv.txHdrLen = C.EthFaceEtherHdr_Init(&priv.txHdr,
		(*C.struct_rte_ether_addr)(port.cfg.Local.Ptr()),
		(*C.struct_rte_ether_addr)(face.loc.Remote.Ptr()),
		C.uint16_t(vlan[0]), C.uint16_t(vlan[1]))

	faceC := face.ptr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.EthFace_TxBurst)

	face.FinishInitFaceBase(port.cfg.TxqPkts, port.cfg.Mtu, int(C.sizeof_struct_rte_ether_hdr))

	if e = face.port.startFace(face, false); e != nil {
		return nil, e
	}

	iface.Put(face)
	return face, nil
}

func (face *EthFace) ptr() *C.Face {
	return (*C.Face)(face.Ptr())
}

func (face *EthFace) getPriv() *C.EthFacePriv {
	return (*C.EthFacePriv)(C.Face_GetPriv(face.ptr()))
}

func (face *EthFace) Port() *Port {
	return face.port
}

func (face *EthFace) Locator() iface.Locator {
	return face.loc
}

func (face *EthFace) Close() error {
	face.BeforeClose()
	face.port.stopFace(face)
	face.CloseFaceBase()
	return nil
}

func (face *EthFace) ListRxGroups() []iface.RxGroup {
	switch impl := face.port.impl.(type) {
	case *rxFlowImpl:
		_, rxf := impl.findQueue(func(rxf *rxFlow) bool { return rxf != nil && rxf.face == face })
		return []iface.RxGroup{rxf}
	case *rxTableImpl:
		return []iface.RxGroup{impl.rxt}
	}
	panic(face.port.impl)
}

type ExCounters struct {
	RxQueue int
}

// EthFace extended counters are available at Port granularity.
// This function provides information to locate relevant fields in EthStats.
func (face *EthFace) ReadExCounters() interface{} {
	var cnt ExCounters
	cnt.RxQueue = int(face.getPriv().rxQueue)
	return cnt
}
