package fetch

/*
#include "../../csrc/fetch/window.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func segStateFromC(c *C.FetchSeg) *SegState {
	return (*SegState)(unsafe.Pointer(c))
}

// Window is a sliding window of segment states.
type Window C.FetchWindow

func (win *Window) ptr() *C.FetchWindow {
	return (*C.FetchWindow)(win)
}

// Init allocates and initializes the FetchWindow.
func (win *Window) Init(capacity int, socket eal.NumaSocket) {
	if capacity < 1 {
		capacity = 65536
	}
	capacity = int(C.rte_align32pow2(C.uint32_t(capacity)))

	c := win.ptr()
	c.array = (*C.FetchSeg)(eal.ZmallocAligned("FetchWindow", capacity*int(C.sizeof_FetchSeg), 1, socket))
	c.capacityMask = C.uint(capacity - 1)
}

// Reset clears the state in the FetchWindow.
func (win *Window) Reset() {
	c := win.ptr()
	c.loPos = 0
	c.loSegNum = 0
	c.hiSegNum = 0
}

// Close deallocates the FetchWindow.
func (win *Window) Close() error {
	eal.Free(win.ptr().array)
	return nil
}

// Get retrieve per-segment state.
func (win *Window) Get(segNum uint64) *SegState {
	return segStateFromC(C.FetchWindow_Get(win.ptr(), C.uint64_t(segNum)))
}

// Append adds per-segment state.
func (win *Window) Append() *SegState {
	return segStateFromC(C.FetchWindow_Append(win.ptr()))
}

// Delete removes per-segment state.
func (win *Window) Delete(segNum uint64) {
	C.FetchWindow_Delete(win.ptr(), C.uint64_t(segNum))
}
