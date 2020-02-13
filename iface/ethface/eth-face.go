package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"encoding/binary"
	"errors"
	"fmt"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type EthFace struct {
	iface.FaceBase
	port   *Port
	remote dpdk.EtherAddr
	rxf    *RxFlow
}

func New(port *Port, loc Locator) (face *EthFace, e error) {
	if !loc.Local.IsZero() && !loc.Local.Equal(port.cfg.Local) {
		return nil, errors.New("port has a different local address")
	}
	switch {
	case loc.Remote.IsZero():
		loc.Remote = ndn.NDN_ETHER_MCAST_ADDR
		fallthrough
	case loc.Remote.IsGroup():
		if face = port.FindFace(nil); face != nil {
			return nil, fmt.Errorf("port has another face %d with a group address", face.GetFaceId())
		}
	case loc.Remote.IsUnicast():
		if face = port.FindFace(&loc.Remote); face != nil {
			return nil, fmt.Errorf("port has another face %d with same unicast address", face.GetFaceId())
		}
	default:
		return nil, fmt.Errorf("invalid MAC address")
	}

	face = new(EthFace)
	if e := face.InitFaceBase(iface.AllocId(iface.FaceKind_Eth), int(C.sizeof_EthFacePriv), port.dev.GetNumaSocket()); e != nil {
		return nil, e
	}
	face.port = port
	face.remote = loc.Remote

	priv := face.getPriv()
	priv.port = C.uint16_t(face.port.dev)
	priv.faceId = C.FaceId(face.GetFaceId())
	port.cfg.Local.CopyToC(unsafe.Pointer(&priv.txHdr.s_addr))
	face.remote.CopyToC(unsafe.Pointer(&priv.txHdr.d_addr))
	var etherType [2]byte
	binary.BigEndian.PutUint16(etherType[:], ndn.NDN_ETHERTYPE)
	C.memcpy(unsafe.Pointer(&priv.txHdr.ether_type), unsafe.Pointer(&etherType[0]), 2)

	faceC := face.getPtr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.EthFace_TxBurst)

	face.FinishInitFaceBase(port.cfg.TxqPkts, port.cfg.Mtu, int(C.sizeof_struct_rte_ether_hdr), port.cfg.Mempools)

	if e = face.port.startFace(face, false); e != nil {
		return nil, e
	}

	iface.Put(face)
	return face, nil
}

func (face *EthFace) getPtr() *C.Face {
	return (*C.Face)(face.GetPtr())
}

func (face *EthFace) getPriv() *C.EthFacePriv {
	return (*C.EthFacePriv)(C.Face_GetPriv(face.getPtr()))
}

func (face *EthFace) GetPort() *Port {
	return face.port
}

func (face *EthFace) GetLocator() iface.Locator {
	var loc Locator
	loc.Scheme = locatorScheme
	loc.Port = face.port.dev.GetName()
	loc.Local = face.port.cfg.Local
	loc.Remote = face.remote
	return loc
}

func (face *EthFace) Close() error {
	face.BeforeClose()
	face.port.stopFace(face)
	face.CloseFaceBase()
	return nil
}

func (face *EthFace) ListRxGroups() []iface.IRxGroup {
	switch impl := face.port.impl.(type) {
	case *rxFlowImpl:
		_, rxf := impl.findQueue(func(rxf *RxFlow) bool { return rxf != nil && rxf.face == face })
		return []iface.IRxGroup{rxf}
	case *rxTableImpl:
		return []iface.IRxGroup{impl.rxt}
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
