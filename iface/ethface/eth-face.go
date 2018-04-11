package ethface

/*
#include "eth-face.h"

void
c_EthFace_RxLoop(EthFace* face, uint16_t burstSize, void* cb, void* cbarg)
{
	EthFace_RxLoop(face, burstSize, (Face_RxCb)cb, cbarg);
}
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

func SizeofHeaderMempoolDataroom() int {
	return int(C.EthTx_GetHeaderMempoolDataroom())
}

type EthFace struct {
	iface.Face
}

func New(port dpdk.EthDev, mempools iface.Mempools) (face EthFace, e error) {
	face.AllocCFace(C.sizeof_EthFace, port.GetNumaSocket())
	res := C.EthFace_Init(face.getPtr(), C.uint16_t(port), (*C.FaceMempools)(mempools.GetPtr()))

	if res != 0 {
		return face, dpdk.Errno(res)
	}

	iface.Put(face)
	return face, nil
}

func (face EthFace) getPtr() *C.EthFace {
	return (*C.EthFace)(face.GetPtr())
}

func (face EthFace) GetPort() dpdk.EthDev {
	return dpdk.EthDev(face.getPtr().port)
}

func (face EthFace) RxLoop(burstSize int, cb unsafe.Pointer, cbarg unsafe.Pointer) {
	C.c_EthFace_RxLoop(face.getPtr(), C.uint16_t(burstSize), cb, cbarg)
}

func (face EthFace) StopRxLoop() error {
	C.EthFace_StopRxLoop(face.getPtr())
	return nil
}

func (face EthFace) ListFacesInRxLoop() []iface.FaceId {
	return []iface.FaceId{face.GetFaceId()}
}
