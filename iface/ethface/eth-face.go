package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"bytes"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/ndn"
)

func SizeofTxHeader() int {
	return int(C.EthFace_SizeofTxHeader())
}

// List RemoteUris of available ports.
func ListPortUris() (a []string) {
	for _, port := range dpdk.ListEthDevs() {
		a = append(a, faceuri.MustMakeEtherUri(port.GetName(), nil, 0).String())
	}
	return a
}

func FindPortByUri(uri string) dpdk.EthDev {
	u, e := faceuri.Parse(uri)
	if e != nil || u.Scheme != "ether" {
		return dpdk.ETHDEV_INVALID
	}
	devName, mac, vid := u.ExtractEther()
	if !bytes.Equal([]byte(mac), []byte(ndn.GetEtherMcastAddr())) || vid != 0 {
		// non-default remote address or VLAN identifier are not supported
		return dpdk.ETHDEV_INVALID
	}
	for _, port := range dpdk.ListEthDevs() {
		if faceuri.CleanEthdevName(port.GetName()) == devName {
			return port
		}
	}
	return dpdk.ETHDEV_INVALID
}

type EthFace struct {
	iface.BaseFace
	nRxThreads int // how many RxProc threads are assigned to RxLoops
}

func New(port dpdk.EthDev, mempools iface.Mempools) (*EthFace, error) {
	var face EthFace
	id := iface.FaceId(iface.FaceKind_Eth<<12) | iface.FaceId(port)
	face.InitBaseFace(id, int(C.sizeof_EthFacePriv), port.GetNumaSocket())

	if ok := C.EthFace_Init(face.getPtr(), (*C.FaceMempools)(mempools.GetPtr())); !ok {
		return nil, dpdk.GetErrno()
	}

	iface.Put(&face)
	return &face, nil
}

func (face *EthFace) getPtr() *C.Face {
	return (*C.Face)(face.GetPtr())
}

func (face *EthFace) getPriv() *C.EthFacePriv {
	return (*C.EthFacePriv)(C.Face_GetPriv(face.getPtr()))
}

func (face *EthFace) GetPort() dpdk.EthDev {
	return dpdk.EthDev(face.GetFaceId() & 0x0FFF)
}

func (face *EthFace) GetLocalUri() *faceuri.FaceUri {
	port := face.GetPort()
	return faceuri.MustMakeEtherUri(port.GetName(), port.GetMacAddr(), 0)
}

func (face *EthFace) GetRemoteUri() *faceuri.FaceUri {
	port := face.GetPort()
	return faceuri.MustMakeEtherUri(port.GetName(), nil, 0)
}

func (face *EthFace) Close() error {
	face.BeforeClose()
	face.CloseBaseFace()
	return nil
}

func (face *EthFace) ReadExCounters() interface{} {
	return face.GetPort().GetStats()
}
