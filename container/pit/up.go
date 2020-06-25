package pit

/*
#include "../../csrc/pcct/pit-up.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Up represents a PIT upstream record.
type Up struct {
	c     *C.PitUp
	entry *Entry
}

// GetFaceId returns the face ID.
func (up Up) GetFaceId() iface.FaceId {
	return iface.FaceId(up.c.face)
}
