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

// FaceID returns the face ID.
func (up Up) FaceID() iface.ID {
	return iface.ID(up.c.face)
}
