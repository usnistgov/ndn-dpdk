package spdkenv

/*
extern void go_SpdkThread_RecvMsg(void* ctx);
extern void go_SpdkThread_InvokePoller(void* ctx);

#include "../../csrc/spdk/thread.h"
*/
import "C"
import (
	"errors"
	"reflect"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

var threadLibInitOnce sync.Once

// Thread represents an SPDK thread.
type Thread struct {
	ealthread.Thread
	c *C.SpdkThread
}

// NewThread creates an SPDK thread.
// The caller needs to assigned it a DPDK lcore and launch it.
func NewThread(name string) (*Thread, error) {
	threadLibInitOnce.Do(func() { C.spdk_thread_lib_init(nil, 0) })

	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	spdkThread := C.spdk_thread_create(nameC, nil)
	if spdkThread == nil {
		return nil, errors.New("spdk_thread_create error")
	}

	th := &Thread{
		c: (*C.SpdkThread)(eal.Zmalloc("SpdkThread", C.sizeof_SpdkThread, eal.NumaSocket{})),
	}
	th.c.spdkTh = spdkThread
	th.Thread = ealthread.New(
		func() int { return int(C.SpdkThread_Run(th.c)) },
		ealthread.InitStopFlag(unsafe.Pointer(&th.c.stop)),
	)
	return th, nil
}

// ThreadRole returns "SPDK" used in lcore allocator.
func (th *Thread) ThreadRole() string {
	return "SPDK"
}

// Ptr return *C.struct_spdk_thread pointer.
func (th *Thread) Ptr() unsafe.Pointer {
	return unsafe.Pointer(th.c.spdkTh)
}

// Close stops the thread and deallocates data structures.
func (th *Thread) Close() error {
	th.Stop()
	eal.Free(th.c)
	return nil
}

// Post asynchronously posts a function to be executed on the SPDK thread.
func (th *Thread) Post(f func()) {
	C.spdk_thread_send_msg(th.c.spdkTh, C.spdk_msg_fn(C.go_SpdkThread_RecvMsg), cptr.CtxPut(f))
}

// Call executes a function on the SPDK thread and waits for its completion.
// f must be a function with zero parameters and zero or one return values.
// Returns f's return value, or nil if f does not have a return value.
func (th *Thread) Call(f interface{}) interface{} {
	done := make(chan interface{})
	th.Post(func() {
		res := reflect.ValueOf(f).Call(nil)
		if len(res) > 0 {
			done <- res[0].Interface()
		} else {
			done <- nil
		}
	})
	return <-done
}

//export go_SpdkThread_RecvMsg
func go_SpdkThread_RecvMsg(ctx unsafe.Pointer) {
	f := cptr.CtxPop(ctx).(func())
	f()
}
