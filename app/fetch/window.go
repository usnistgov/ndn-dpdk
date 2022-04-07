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
// capacity must be power of two.
func (win *Window) Init(capacity int, socket eal.NumaSocket) {
	win.array = eal.ZmallocAligned[C.FetchSeg]("FetchWindow", capacity*C.sizeof_FetchSeg, 1, socket)
	win.capacityMask = C.uint(capacity - 1)
}

// Reset clears the state in the FetchWindow.
func (win *Window) Reset() {
	win.loPos = 0
	win.loSegNum = 0
	win.hiSegNum = 0
}

// Close deallocates the FetchWindow.
func (win *Window) Close() error {
	eal.Free(win.array)
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
