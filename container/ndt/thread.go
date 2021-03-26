package ndt

/*
#include "../../csrc/ndt/ndt.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"inet.af/netstack/gohacks"
)

// Thread represents an NDT lookup thread.
type Thread C.NdtThread

// Ptr returns *C.NdtThread pointer.
func (ndtt *Thread) Ptr() unsafe.Pointer {
	return unsafe.Pointer(ndtt.ptr())
}

func (ndtt *Thread) ptr() *C.NdtThread {
	return (*C.NdtThread)(ndtt)
}

// Close releases memory.
func (ndtt *Thread) Close() error {
	eal.Free(ndtt.ptr())
	return nil
}

// Lookup queries a name and increments hit counters.
func (ndtt *Thread) Lookup(name ndn.Name) uint8 {
	nameP := ndni.NewPName(name)
	defer nameP.Free()
	return uint8(C.Ndtt_Lookup(ndtt.ptr(), (*C.PName)(nameP.Ptr())))
}

func (ndtt *Thread) hitCounters(nEntries int) (hits []uint32) {
	sh := (*gohacks.SliceHeader)(unsafe.Pointer(&hits))
	sh.Data = unsafe.Pointer(C.Ndtt_Hits_(ndtt.ptr()))
	sh.Len = nEntries
	sh.Cap = nEntries
	return hits
}

func newThread(ndt *Ndt, socket eal.NumaSocket) *Thread {
	c := C.Ndtt_New_(ndt.replicas[socket].ptr(), C.int(socket.ID()))
	return (*Thread)(c)
}
