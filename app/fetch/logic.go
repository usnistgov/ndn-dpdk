package fetch

/*
#include "../../csrc/fetch/logic.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"unsafe"
)

// Logic implements fetcher congestion control and scheduling logic.
type Logic C.FetchLogic

// LogicFromPtr converts *C.FetchLogic to *Logic.
// ptr must be in C memory due to TAILQ_HEAD usage.
func LogicFromPtr(ptr unsafe.Pointer) (fl *Logic) {
	return (*Logic)(ptr)
}

func (fl *Logic) ptr() *C.FetchLogic {
	return (*C.FetchLogic)(unsafe.Pointer(fl))
}

// Window returns the segment state window.
func (fl *Logic) Window() *Window {
	return (*Window)(&fl.ptr().win)
}

// RttEst returns the RTT estimator.
func (fl *Logic) RttEst() *RttEst {
	return (*RttEst)(&fl.ptr().rtte)
}

// Cubic returns the congestion avoidance algorithm.
func (fl *Logic) Cubic() *Cubic {
	return (*Cubic)(&fl.ptr().ca)
}

// Init initializes the logic and allocates data structures.
func (fl *Logic) Init(windowCapacity int, socket eal.NumaSocket) {
	fl.Window().Init(windowCapacity, socket)
	fl.RttEst().Init()
	fl.Cubic().Init()
	C.FetchLogic_Init_(fl.ptr())
}

// Reset resets this to initial state.
func (fl *Logic) Reset() {
	c := fl.ptr()
	C.MinSched_Close(c.sched)
	*c = C.FetchLogic{win: c.win}
	fl.Window().Reset()
	fl.RttEst().Init()
	fl.Cubic().Init()
	C.FetchLogic_Init_(fl.ptr())
}

// Close deallocates data structures.
func (fl *Logic) Close() error {
	C.MinSched_Close(fl.ptr().sched)
	return fl.Window().Close()
}

// SetFinalSegNum assigns (inclusive) final segment number.
func (fl *Logic) SetFinalSegNum(segNum uint64) {
	C.FetchLogic_SetFinalSegNum(fl.ptr(), C.uint64_t(segNum))
}

// Finished determines if all segments have been fetched.
func (fl *Logic) Finished() bool {
	return bool(C.FetchLogic_Finished(fl.ptr()))
}

// TriggerRtoSched triggers the internal RTO scheduler.
func (fl *Logic) TriggerRtoSched() {
	C.MinSched_Trigger(fl.ptr().sched)
}

// TxInterest requests for Interest transmission.
func (fl *Logic) TxInterest() (need bool, segNum uint64) {
	var segNumC C.uint64_t
	n := C.FetchLogic_TxInterestBurst(fl.ptr(), &segNumC, 1)
	return n > 0, uint64(segNumC)
}

// RxData notifies about Data arrival.
func (fl *Logic) RxData(segNum uint64, hasCongMark bool) {
	var pkt C.FetchLogicRxData
	pkt.segNum = C.uint64_t(segNum)
	if hasCongMark {
		pkt.congMark = 1
	}
	C.FetchLogic_RxDataBurst(fl.ptr(), &pkt, 1)
}
