package fetch

/*
#include "logic.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

// Allocate *Logic in C memory.
// It's unsafe to use Logic in Go memory due to TAILQ_HEAD usage.
// Init() must be called separately.
func NewLogic() (fl *Logic) {
	return LogicFromPtr(dpdk.Zmalloc("FetchLogic", C.sizeof_FetchLogic, dpdk.NUMA_SOCKET_ANY))
}

// Convert *C.FetchLogic to *Logic.
func LogicFromPtr(ptr unsafe.Pointer) (fl *Logic) {
	return (*Logic)(ptr)
}

func (fl *Logic) getPtr() *C.FetchLogic {
	return (*C.FetchLogic)(unsafe.Pointer(fl))
}

func (fl *Logic) Init(windowCapacity int, socket dpdk.NumaSocket) {
	fl.Win.Init(windowCapacity, socket)
	fl.Rtte.Init()
	fl.Ca.Init()
	C.FetchLogic_Init_(fl.getPtr())
}

func (fl *Logic) Close() error {
	C.MinSched_Close(fl.getPtr().sched)
	return fl.Win.Close()
}

// Deallocate. Use only if this was allocated via NewLogic().
func (fl *Logic) CloseAndFree() {
	fl.Close()
	dpdk.Free(unsafe.Pointer(fl))
}

// Set (inclusive) final segment number.
func (fl *Logic) SetFinalSegNum(segNum uint64) {
	C.FetchLogic_SetFinalSegNum(fl.getPtr(), C.uint64_t(segNum))
}

// Determine if all segments have been fetched.
func (fl *Logic) Finished() bool {
	return bool(C.FetchLogic_Finished(fl.getPtr()))
}

// Trigger the internal RTO scheduler.
func (fl *Logic) TriggerRtoSched() {
	C.MinSched_Trigger(fl.getPtr().sched)
}

// Request for Interest transmission.
func (fl *Logic) TxInterest() (need bool, segNum uint64) {
	var segNumC C.uint64_t
	n := C.FetchLogic_TxInterestBurst(fl.getPtr(), &segNumC, 1)
	return n > 0, uint64(segNumC)
}

// Notify Data arrival.
func (fl *Logic) RxData(segNum uint64, hasCongMark bool) {
	var pkt C.FetchLogicRxData
	pkt.segNum = C.uint64_t(segNum)
	if hasCongMark {
		pkt.congMark = 1
	}
	C.FetchLogic_RxDataBurst(fl.getPtr(), &pkt, 1)
}
