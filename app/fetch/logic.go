package fetch

/*
#include "../../csrc/fetch/logic.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// Logic implements fetcher congestion control and scheduling logic.
type Logic C.FetchLogic

func (fl *Logic) ptr() *C.FetchLogic {
	return (*C.FetchLogic)(fl)
}

func (fl *Logic) window() *Window {
	return (*Window)(&fl.win)
}

// Init initializes the logic and allocates data structures.
func (fl *Logic) Init(windowCapacity int, socket eal.NumaSocket) {
	fl.window().Init(windowCapacity, socket)
	C.RttEst_Init(&fl.rtte)
	C.TcpCubic_Init(&fl.ca)
	C.FetchLogic_Init_(fl.ptr())
}

// Reset resets this to initial state.
func (fl *Logic) Reset() {
	C.MinSched_Close(fl.sched)
	*fl = Logic{win: fl.win}
	fl.window().Reset()
	C.RttEst_Init(&fl.rtte)
	C.TcpCubic_Init(&fl.ca)
	C.FetchLogic_Init_(fl.ptr())
}

// Close deallocates data structures.
func (fl *Logic) Close() error {
	C.MinSched_Close(fl.sched)
	fl.window().Close()
	return nil
}

// SetFinalSegNum assigns (inclusive) final segment number.
func (fl *Logic) SetFinalSegNum(segNum uint64) {
	C.FetchLogic_SetFinalSegNum(fl.ptr(), C.uint64_t(segNum))
}

// Finished determines if all segments have been fetched.
func (fl *Logic) Finished() bool {
	return bool(C.FetchLogic_Finished(fl.ptr()))
}
