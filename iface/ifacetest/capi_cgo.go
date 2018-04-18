package ifacetest

/*
#include "../face.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func Face_IsDown(faceId iface.FaceId) bool {
	return bool(C.Face_IsDown(C.FaceId(faceId)))
}

func Face_TxBurst(faceId iface.FaceId, pkts []ndn.Packet) {
	C.Face_TxBurst(C.FaceId(faceId), (**C.Packet)(unsafe.Pointer(&pkts[0])), C.uint16_t(len(pkts)))
}
