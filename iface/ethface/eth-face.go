package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"encoding/binary"
	"net"
	"unsafe"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/ndn"
)

type faceFactory struct {
	Port     *Port
	Mempools iface.Mempools
	Local    net.HardwareAddr
	Mtu      int
	TxqPkts  int
	Flows    map[iface.FaceId]*RxFlow
}

func copyHwaddrToC(a net.HardwareAddr, c *C.struct_ether_addr) {
	for i := 0; i < C.ETHER_ADDR_LEN; i++ {
		c.addr_bytes[i] = C.uint8_t(a[i])
	}
}

func (f *faceFactory) NewFace(id iface.FaceId, remote net.HardwareAddr) (face *EthFace, e error) {
	face = new(EthFace)
	if e := face.InitBaseFace(id, int(C.sizeof_EthFacePriv), f.Port.GetNumaSocket()); e != nil {
		return nil, e
	}

	face.port = f.Port
	face.local = f.Local
	if remote == nil {
		face.remote = ndn.GetEtherMcastAddr()
	} else {
		face.remote = remote
	}
	if f.Flows != nil {
		face.rxf = f.Flows[id]
	}

	priv := face.getPriv()
	priv.port = C.uint16_t(face.port.dev)
	copyHwaddrToC(face.local, &priv.txHdr.s_addr)
	copyHwaddrToC(face.remote, &priv.txHdr.d_addr)

	var etherType [2]byte
	binary.BigEndian.PutUint16(etherType[:], ndn.NDN_ETHERTYPE)
	C.memcpy(unsafe.Pointer(&priv.txHdr.ether_type), unsafe.Pointer(&etherType[0]), 2)

	faceC := face.getPtr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.EthFace_TxBurst)

	face.FinishInitBaseFace(f.TxqPkts, f.Mtu, int(C.sizeof_struct_ether_hdr), f.Mempools)
	iface.Put(face)
	return face, nil
}

type EthFace struct {
	iface.BaseFace
	port   *Port
	local  net.HardwareAddr
	remote net.HardwareAddr
	rxf    *RxFlow
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

func (face *EthFace) GetLocalUri() *faceuri.FaceUri {
	return faceuri.MustMakeEtherUri(face.port.dev.GetName(), face.local, 0)
}

func (face *EthFace) GetRemoteUri() *faceuri.FaceUri {
	return faceuri.MustMakeEtherUri(face.port.dev.GetName(), face.remote, 0)
}

func (face *EthFace) Close() error {
	face.BeforeClose()
	if face.rxf != nil {
		face.rxf.Close()
	}
	if face.port.multicast == face {
		face.port.multicast = nil
	} else {
		for i, entry := range face.port.unicast {
			if entry == face {
				face.port.unicast[i] = face.port.unicast[len(face.port.unicast)-1]
				face.port.unicast = face.port.unicast[:len(face.port.unicast)-1]
				break
			}
		}
	}
	face.CloseBaseFace()
	return nil
}

func (face *EthFace) ListRxGroups() []iface.IRxGroup {
	if face.rxf != nil {
		return []iface.IRxGroup{face.rxf}
	}
	return face.port.ListRxGroups()
}

func (face *EthFace) ReadExCounters() interface{} {
	return face.port.dev.GetStats()
}
