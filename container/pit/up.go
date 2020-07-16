package pit

/*
#include "../../csrc/pcct/pit-up.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/iface"
)

// UpRecord represents a PIT upstream record.
type UpRecord struct {
	c     *C.PitUp
	entry *Entry
}

// FaceID returns the face ID.
func (up UpRecord) FaceID() iface.ID {
	return iface.ID(up.c.face)
}
