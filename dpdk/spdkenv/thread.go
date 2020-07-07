package spdkenv

/*
#include "../../csrc/dpdk/spdk-thread.h"
*/
import "C"
import (
	"errors"
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
		cptr.CFunction(unsafe.Pointer(C.SpdkThread_Run), unsafe.Pointer(th.c)),
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

func (th *Thread) main() {
	C.SpdkThread_Run(th.c)
}

// Post asynchronously posts a function to be executed on the SPDK thread.
func (th *Thread) Post(fn cptr.Function) {
	f, arg := fn.MakeCFunction()
	C.spdk_thread_send_msg(th.c.spdkTh, C.spdk_msg_fn(f), arg)
}
