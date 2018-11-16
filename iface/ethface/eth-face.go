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
	port     *Port
	mempools iface.Mempools
	local    net.HardwareAddr
	mtu      int
}

func copyHwaddrToC(a net.HardwareAddr, c *C.struct_ether_addr) {
	for i := 0; i < C.ETHER_ADDR_LEN; i++ {
		c.addr_bytes[i] = C.uint8_t(a[i])
	}
}

func (f *faceFactory) newFace(id iface.FaceId, remote net.HardwareAddr) (face *EthFace) {
	face = new(EthFace)
	face.InitBaseFace(id, int(C.sizeof_EthFacePriv), f.port.GetNumaSocket())
	face.port = f.port
	face.local = f.local
	if remote == nil {
		face.remote = ndn.GetEtherMcastAddr()
	} else {
		face.remote = remote
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
	C.FaceImpl_Init(faceC, C.uint16_t(f.mtu), C.sizeof_struct_ether_hdr,
		(*C.FaceMempools)(f.mempools.GetPtr()))
	iface.Put(face)
	return face
}

type EthFace struct {
	iface.BaseFace
	port   *Port
	local  net.HardwareAddr
	remote net.HardwareAddr
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
	face.CloseBaseFace()
	return nil
}

func (face *EthFace) ReadExCounters() interface{} {
	return face.port.dev.GetStats()
}
