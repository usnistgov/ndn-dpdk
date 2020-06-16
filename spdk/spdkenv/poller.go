package spdkenv

/*
extern void go_SpdkThread_RecvMsg(void* ctx);
extern void go_SpdkThread_InvokePoller(void* ctx);

#include "../../csrc/spdk/thread.h"
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
	poller.th.Call(func() {
		poller.c = C.spdk_poller_register(f, arg, C.uint64_t(d/time.Microsecond))
	})
}

//export go_SpdkThread_InvokePoller
func go_SpdkThread_InvokePoller(ctx unsafe.Pointer) {
	f := cptr.CtxGet(ctx).(func())
	f()
}

// Stop cancels periodical execution.
func (poller *Poller) Stop() {
	poller.th.Call(func() { C.spdk_poller_unregister(&poller.c) })
	if poller.ctx != nil {
		cptr.CtxClear(poller.ctx)
		poller.ctx = nil
	}
}
