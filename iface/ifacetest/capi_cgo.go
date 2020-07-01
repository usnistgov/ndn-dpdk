package ifacetest

/*
#include "../../csrc/iface/face.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func Face_IsDown(faceID iface.ID) bool {
	return bool(C.Face_IsDown(C.FaceID(faceID)))
}

func Face_TxBurst(faceID iface.ID, pkts []*ndni.Packet) {
	ptr, count := cptr.ParseCptrArray(pkts)
	C.Face_TxBurst(C.FaceID(faceID), (**C.Packet)(ptr), C.uint16_t(count))
}
