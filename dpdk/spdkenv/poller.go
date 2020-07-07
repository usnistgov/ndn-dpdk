package spdkenv

/*
#include "../../csrc/dpdk/spdk-thread.h"

extern int go_SpdkThread_InvokePoller(void* ctx);
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

// Poller periodically executes a function on an SPDK thread.
type Poller struct {
	c   *C.struct_spdk_poller
	th  *Thread
	ctx unsafe.Pointer
}

// NewPoller creates a Poller from a Go function.
func NewPoller(th *Thread, f func(), d time.Duration) (poller *Poller) {
	poller = new(Poller)
	poller.th = th
	poller.ctx = cptr.CtxPut(f)
	poller.start(C.spdk_poller_fn(C.go_SpdkThread_InvokePoller), poller.ctx, d)
	return poller
}

// NewPollerC creates a Poller from a C function.
func NewPollerC(th *Thread, f, arg unsafe.Pointer, d time.Duration) (poller *Poller) {
	poller = new(Poller)
	poller.th = th
	poller.start(C.spdk_poller_fn(f), arg, d)
	return poller
}

func (poller *Poller) start(f C.spdk_poller_fn, arg unsafe.Pointer, d time.Duration) {
	cptr.Call(poller.th.Post, func() {
		poller.c = C.spdk_poller_register(f, arg, C.uint64_t(d/time.Microsecond))
	})
}

//export go_SpdkThread_InvokePoller
func go_SpdkThread_InvokePoller(ctx unsafe.Pointer) C.int {
	f := cptr.CtxGet(ctx).(func())
	f()
	return -1
}

// Stop cancels periodical execution.
func (poller *Poller) Stop() {
	cptr.Call(poller.th.Post, func() { C.spdk_poller_unregister(&poller.c) })
	if poller.ctx != nil {
		cptr.CtxClear(poller.ctx)
		poller.ctx = nil
	}
}
