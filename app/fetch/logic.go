package fetch

/*
#include "../../csrc/fetch/logic.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"unsafe"
)

// Convert *C.FetchLogic to *Logic.
// ptr must be in C memory due to TAILQ_HEAD usage.
func LogicFromPtr(ptr unsafe.Pointer) (fl *Logic) {
	return (*Logic)(ptr)
}

func (fl *Logic) ptr() *C.FetchLogic {
	return (*C.FetchLogic)(unsafe.Pointer(fl))
}

func (fl *Logic) Init(windowCapacity int, socket eal.NumaSocket) {
	fl.Win.Init(windowCapacity, socket)
	fl.Rtte.Init()
	fl.Ca.Init()
	C.FetchLogic_Init_(fl.ptr())
}

// Reset to initial state.
func (fl *Logic) Reset() {
	C.MinSched_Close(fl.ptr().sched)
	*fl = Logic{Win: fl.Win}
	fl.Win.Reset()
	fl.Rtte.Init()
	fl.Ca.Init()
	C.FetchLogic_Init_(fl.ptr())
}

func (fl *Logic) Close() error {
	C.MinSched_Close(fl.ptr().sched)
	return fl.Win.Close()
}

// Set (inclusive) final segment number.
func (fl *Logic) SetFinalSegNum(segNum uint64) {
	C.FetchLogic_SetFinalSegNum(fl.ptr(), C.uint64_t(segNum))
}

// Determine if all segments have been fetched.
func (fl *Logic) Finished() bool {
	return bool(C.FetchLogic_Finished(fl.ptr()))
}

// Trigger the internal RTO scheduler.
func (fl *Logic) TriggerRtoSched() {
	C.MinSched_Trigger(fl.ptr().sched)
}

// Request for Interest transmission.
func (fl *Logic) TxInterest() (need bool, segNum uint64) {
	var segNumC C.uint64_t
	n := C.FetchLogic_TxInterestBurst(fl.ptr(), &segNumC, 1)
	return n > 0, uint64(segNumC)
}

// Notify Data arrival.
func (fl *Logic) RxData(segNum uint64, hasCongMark bool) {
	var pkt C.FetchLogicRxData
	pkt.segNum = C.uint64_t(segNum)
	if hasCongMark {
		pkt.congMark = 1
	}
	C.FetchLogic_RxDataBurst(fl.ptr(), &pkt, 1)
}
