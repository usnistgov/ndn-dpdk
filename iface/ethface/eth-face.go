package ethface

/*
#include "eth-face.h"
*/
import "C"
import (
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

func SizeofHeaderMempoolDataRoom() uint16 {
	return uint16(C.EthTx_GetHeaderMempoolDataRoom())
}

type EthFace struct {
	iface.Face
}

func New(port dpdk.EthDev, indirectMp dpdk.PktmbufPool,
	headerMp dpdk.PktmbufPool) (face EthFace, e error) {
	face = EthFace{iface.FaceFromPtr(C.calloc(1, C.sizeof_EthFace))}
	res := C.EthFace_Init(face.getPtr(), C.uint16_t(port),
		(*C.struct_rte_mempool)(indirectMp.GetPtr()), (*C.struct_rte_mempool)(headerMp.GetPtr()))

	if res != 0 {
		return face, dpdk.Errno(res)
	}
	return face, nil
}

func (face EthFace) getPtr() *C.EthFace {
	return (*C.EthFace)(face.GetPtr())
}
