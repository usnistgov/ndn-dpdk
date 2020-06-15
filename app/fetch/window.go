package fetch

/*
#include "../../csrc/fetch/window.h"
*/
import "C"
import (
	"ndn-dpdk/dpdk/eal"
	"unsafe"
)

func segStateFromC(c *C.FetchSeg) *SegState {
	return (*SegState)(unsafe.Pointer(c))
}

func WindowFromPtr(ptr unsafe.Pointer) (win *Window) {
	return (*Window)(ptr)
}

func (win *Window) getPtr() *C.FetchWindow {
	return (*C.FetchWindow)(unsafe.Pointer(win))
}

func (win *Window) Init(capacity int, socket eal.NumaSocket) {
	if capacity < 1 {
		capacity = 65536
	}
	capacity = int(C.rte_align32pow2(C.uint32_t(capacity)))

	win.Array = (*SegState)(eal.ZmallocAligned("FetchWindow", capacity*int(C.sizeof_FetchSeg), 1, socket))
	win.CapacityMask = uint32(capacity - 1)
}

func (win *Window) Reset() {
	win.LoPos = 0
	win.LoSegNum = 0
	win.HiSegNum = 0
}

func (win *Window) Close() error {
	eal.Free(win.Array)
	return nil
}

func (win *Window) Get(segNum uint64) *SegState {
	return segStateFromC(C.FetchWindow_Get(win.getPtr(), C.uint64_t(segNum)))
}

func (win *Window) Append() *SegState {
	return segStateFromC(C.FetchWindow_Append(win.getPtr()))
}

func (win *Window) Delete(segNum uint64) {
	C.FetchWindow_Delete(win.getPtr(), C.uint64_t(segNum))
}
