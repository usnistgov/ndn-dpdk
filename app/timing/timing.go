package timing

/*
#include "timing.h"
*/
import "C"
import (
	"log"
	"unsafe"

	"ndn-dpdk/dpdk"
)

func Init(ringCapacity int) error {
	r, e := dpdk.NewRing("gTimingRing", ringCapacity, dpdk.NUMA_SOCKET_ANY, false, true)
	if e != nil {
		return e
	}
	C.gTimingRing = (*C.struct_rte_ring)(r.GetPtr())

	return nil
}

type Writer struct {
	Filename string
	NSkip    int
	NTotal   int
}

func (w Writer) Run() int {
	filenameC := C.CString(w.Filename)
	defer C.free(unsafe.Pointer(filenameC))
	res := int(C.Timing_RunWriter(filenameC, C.int(w.NSkip), C.int(w.NTotal)))
	log.Printf("TWRES=%d", res)
	return res
}
