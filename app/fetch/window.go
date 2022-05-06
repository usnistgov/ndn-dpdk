package fetch

/*
#include "../../csrc/fetch/window.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// Window is a sliding window of segment states.
type Window C.FetchWindow

// Init allocates and initializes the FetchWindow.
// capacity must be power of two.
func (win *Window) Init(capacity int, socket eal.NumaSocket) {
	win.array = eal.ZmallocAligned[C.FetchSeg]("FetchWindow", capacity*C.sizeof_FetchSeg, 1, socket)
	win.capacityMask = C.uint(capacity - 1)
}

// Reset clears the state in the FetchWindow.
func (win *Window) Reset() {
	win.loPos, win.loSegNum, win.hiSegNum = 0, 0, 0
}

// Close deallocates the FetchWindow.
func (win *Window) Close() error {
	eal.Free(win.array)
	return nil
}
