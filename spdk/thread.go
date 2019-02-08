package spdk

/*
extern void go_SpdkThread_RecvMsg(void* ctx);
extern void go_SpdkThread_InvokePoller(void* ctx);

#include "thread.h"
*/
import "C"
import (
	"errors"
	"reflect"
	"sync"
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
)

var threadLibInitOnce sync.Once

// SPDK thread.
type Thread struct {
	dpdk.ThreadBase
	c *C.SpdkThread
}

// Create an SPDK thread.
// It needs to be assigned to a DPDK lcore and launched.
func NewThread(name string) (th *Thread, e error) {
	threadLibInitOnce.Do(func() { C.spdk_thread_lib_init(nil) })

	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	spdkThread := C.spdk_thread_create(nameC)
	if spdkThread == nil {
		return nil, errors.New("spdk_thread_create error")
	}

	th = new(Thread)
	th.ResetThreadBase()
	th.c = (*C.SpdkThread)(dpdk.Zmalloc("SpdkThread", C.sizeof_SpdkThread, dpdk.NUMA_SOCKET_ANY))
	th.c.spdkTh = spdkThread
	dpdk.InitStopFlag(unsafe.Pointer(&th.c.stop))
	return th, nil
}

// Get native *C.struct_spdk_thread pointer to use in other packages.
func (th *Thread) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(th.c.spdkTh)
}

func (th *Thread) Launch() error {
	return th.LaunchImpl(func() int {
		C.SpdkThread_Run(th.c)
		return 0
	})
}

func (th *Thread) Stop() error {
	return th.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&th.c.stop)))
}

func (th *Thread) Close() error {
	if th.IsRunning() {
		return dpdk.ErrCloseRunningThread
	}
	dpdk.Free(th.c)
	return nil
}

// Asynchronously post a function to be executed on the SPDK thread.
func (th *Thread) Post(f func()) {
	C.spdk_thread_send_msg(th.c.spdkTh, C.spdk_msg_fn(C.go_SpdkThread_RecvMsg), ctxPut(f))
}

// Execute a function on the SPDK thread and wait for its completion.
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
	f := ctxPop(ctx).(func())
	f()
}

// SPDK poller on a thread.
type Poller struct {
	c   *C.struct_spdk_poller
	th  *Thread
	ctx unsafe.Pointer
}

// Periodically execute a function on an SPDK thread.
func NewPoller(th *Thread, f func(), d time.Duration) (poller *Poller) {
	poller = new(Poller)
	poller.th = th
	poller.ctx = ctxPut(f)
	poller.start(C.spdk_poller_fn(C.go_SpdkThread_InvokePoller), poller.ctx, d)
	return poller
}

// Periodically execute a C function on an SPDK thread.
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
	f := ctxGet(ctx).(func())
	f()
}

// Cancel periodical execution.
func (poller *Poller) Stop() {
	poller.th.Call(func() { C.spdk_poller_unregister(&poller.c) })
	if poller.ctx != nil {
		ctxClear(poller.ctx)
		poller.ctx = nil
	}
}
