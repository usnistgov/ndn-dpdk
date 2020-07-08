package ifacetest

/*
#include "../../csrc/iface/face.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/iface"
)

func Face_IsDown(faceID iface.ID) bool {
	return bool(C.Face_IsDown(C.FaceID(faceID)))
}
