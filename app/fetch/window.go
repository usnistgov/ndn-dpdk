package fetch

/*
#include "window.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

func fetchSegFromC(c *C.FetchSeg) *FetchSeg {
	return (*FetchSeg)(unsafe.Pointer(c))
}

func fetchWindowFromC(c *C.FetchWindow) (win *FetchWindow) {
	return (*FetchWindow)(unsafe.Pointer(c))
}

func (win *FetchWindow) getPtr() *C.FetchWindow {
	return (*C.FetchWindow)(unsafe.Pointer(win))
}

func (win *FetchWindow) Init(capacity int, socket dpdk.NumaSocket) {
	win.Array = (*FetchSeg)(dpdk.ZmallocAligned("FetchWindow", capacity*int(C.sizeof_FetchSeg), 1, socket))
	win.CapacityMask = uint32(capacity - 1)
}

func (win *FetchWindow) Close() error {
	dpdk.Free(win.Array)
	return nil
}

func (win *FetchWindow) Get(segNum uint64) *FetchSeg {
	return fetchSegFromC(C.FetchWindow_Get(win.getPtr(), C.uint64_t(segNum)))
}

func (win *FetchWindow) Append() (segNum uint64, seg *FetchSeg) {
	seg = fetchSegFromC(C.FetchWindow_Append(win.getPtr(), (*C.uint64_t)(unsafe.Pointer(&segNum))))
	return
}

func (win *FetchWindow) Delete(segNum uint64) {
	C.FetchWindow_Delete(win.getPtr(), C.uint64_t(segNum))
}
