package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"unsafe"

	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type EthFace struct {
	iface.FaceBase
	port   *Port
	remote net.HardwareAddr
	rxf    *RxFlow
}

func New(port *Port, loc Locator) (face *EthFace, e error) {
	if loc.Local != nil && bytes.Compare(([]byte)(port.cfg.Local), ([]byte)(loc.Local)) != 0 {
		return nil, errors.New("conflicting local address")
	}
	if loc.Remote == nil {
		loc.Remote = ndn.GetEtherMcastAddr()
	}
	switch classifyMac48(loc.Remote) {
	case mac48_multicast:
		if face = port.FindFace(nil); face != nil {
			return nil, fmt.Errorf("face %d has multicast address", face.GetFaceId())
		}
	case mac48_unicast:
		if face = port.FindFace(loc.Remote); face != nil {
			return nil, fmt.Errorf("face %d has same unicast address", face.GetFaceId())
		}
	default:
		return nil, fmt.Errorf("invalid MAC-48 address")
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
	copyMac48ToC(port.cfg.Local, &priv.txHdr.s_addr)
	copyMac48ToC(face.remote, &priv.txHdr.d_addr)
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
