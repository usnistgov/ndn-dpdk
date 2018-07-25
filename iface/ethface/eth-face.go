package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"fmt"
	"strings"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
)

func SizeofTxHeader() int {
	return int(C.EthFace_SizeofTxHeader())
}

// List RemoteUris of available ports.
func ListPortUris() (a []string) {
	for _, port := range dpdk.ListEthDevs() {
		a = append(a, makeRemoteUri(port))
	}
	return a
}

func FindPortByUri(uri string) dpdk.EthDev {
	for _, port := range dpdk.ListEthDevs() {
		if makeRemoteUri(port) == uri {
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
	return faceuri.MustParse(fmt.Sprintf("ether://[%s]", face.GetPort().GetMacAddr()))
}

func makeRemoteUri(port dpdk.EthDev) string {
	hostname := strings.Map(func(c rune) rune {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			return c
		}
		return '-'
	}, port.GetName())
	return fmt.Sprintf("dev://%s", hostname)
}

func (face *EthFace) GetRemoteUri() *faceuri.FaceUri {
	return faceuri.MustParse(makeRemoteUri(face.GetPort()))
}

func (face *EthFace) Close() error {
	face.BeforeClose()
	C.EthFace_Close(face.getPtr())
	face.CloseBaseFace()
	return nil
}

func (face *EthFace) ReadExCounters() interface{} {
	return face.GetPort().GetStats()
}
