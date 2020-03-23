package hrlog

/*
#include "writer.h"
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"

	"ndn-dpdk/dpdk"
)

const ringCapacity = 65536

// Initialize high resolution logger.
func Init() error {
	r, e := dpdk.NewRing("theHrlogRing", ringCapacity, dpdk.NUMA_SOCKET_ANY, false, true)
	if e != nil {
		return e
	}
	C.theHrlogRing = (*C.struct_rte_ring)(r.GetPtr())
	return nil
}

// Management module for high resolution logger.
type HrlogMgmt struct{}

var collectLock sync.Mutex

func (HrlogMgmt) Collect(args CollectArgs, reply *struct{}) error {
	collectLock.Lock()
	defer collectLock.Unlock()
	filenameC := C.CString(args.Filename)
	defer C.free(unsafe.Pointer(filenameC))
	res := C.Hrlog_RunWriter(filenameC, ringCapacity, C.int(args.Count))
	if res != 0 {
		return fmt.Errorf("Hrlog_RunWriter error %d", res)
	}
	return nil
}

type CollectArgs struct {
	Filename string
	Count    int
}
