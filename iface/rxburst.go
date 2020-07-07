package iface

/*
#include "../csrc/iface/rxburst.h"

void go_Face_RxBurstCallback(FaceRxBurst* burst, void* arg);
*/
import "C"
import (
	"io"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// RxBurst stores a burst of received packets.
type RxBurst C.FaceRxBurst

// NewRxBurst allocates an RxBurst.
// The capacity is for each L3 packet type.
func NewRxBurst(capacity int) (burst *RxBurst) {
	c := C.FaceRxBurst_New(C.uint16_t(capacity))
	return (*RxBurst)(c)
}

// Ptr returns *C.FaceRxBurst pointer.
func (burst *RxBurst) Ptr() unsafe.Pointer {
	return unsafe.Pointer(burst)
}

func (burst *RxBurst) ptr() *C.FaceRxBurst {
	return (*C.FaceRxBurst)(burst)
}

// Close deallocates this RxBurst.
func (burst *RxBurst) Close() error {
	C.FaceRxBurst_Close(burst.ptr())
	return nil
}

// Capacity returns the capacity for each L3 packet type.
func (burst *RxBurst) Capacity() int {
	return int(burst.ptr().capacity)
}

// ListInterests returns Interest packets in the burst.
func (burst *RxBurst) ListInterests() (list []*ndni.Interest) {
	c := burst.ptr()
	list = make([]*ndni.Interest, int(c.nInterests))
	for i := range list {
		npkt := ndni.PacketFromPtr(unsafe.Pointer(C.FaceRxBurst_GetInterest(c, C.uint16_t(i))))
		list[i] = npkt.AsInterest()
	}
	return list
}

// ListData returns Data packets in the burst.
func (burst *RxBurst) ListData() (list []*ndni.Data) {
	c := burst.ptr()
	list = make([]*ndni.Data, int(c.nData))
	for i := range list {
		npkt := ndni.PacketFromPtr(unsafe.Pointer(C.FaceRxBurst_GetData(c, C.uint16_t(i))))
		list[i] = npkt.AsData()
	}
	return list
}

// ListNacks returns Nack packets in the burst.
func (burst *RxBurst) ListNacks() (list []*ndni.Nack) {
	c := burst.ptr()
	list = make([]*ndni.Nack, int(c.nNacks))
	for i := range list {
		npkt := ndni.PacketFromPtr(unsafe.Pointer(C.FaceRxBurst_GetNack(c, C.uint16_t(i))))
		list[i] = npkt.AsNack()
	}
	return list
}

// RxBurstCallback is a callback function that accepts RxBurst.
type RxBurstCallback func(burst *RxBurst)

// WrapRxBurstCallback converts a Go func into *C.Face_RxCb and void* argument.
// cancel.Close() deletes the context, after which the callback panics.
func WrapRxBurstCallback(fn RxBurstCallback) (f, arg unsafe.Pointer, cancel io.Closer) {
	ctx := cptr.CtxPut(fn)
	return unsafe.Pointer(C.go_Face_RxBurstCallback), ctx, cptr.CtxCloser(ctx)
}

//export go_Face_RxBurstCallback
func go_Face_RxBurstCallback(burst *C.FaceRxBurst, ctx unsafe.Pointer) {
	fn := cptr.CtxGet(ctx).(RxBurstCallback)
	fn((*RxBurst)(burst))
}
