package spdkenv

/*
#include "../../csrc/dpdk/spdk-thread.h"

// workaround gopls "compiler(InvalidCall)" false positive
int c_SpdkThread_Run(SpdkThread* th) { return SpdkThread_Run(th); }
*/
import "C"
import (
	"errors"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"go.uber.org/zap"
)

var threadLibInitOnce sync.Once

// Thread represents an SPDK thread.
type Thread struct {
	ealthread.Thread
	c           *C.SpdkThread
	RcuReadSide *urcu.ReadSide
}

var _ eal.PollThread = (*Thread)(nil)
var _ ealthread.ThreadWithRole = (*Thread)(nil)

// NewThread creates an SPDK thread.
// The caller needs to assigned it a DPDK lcore and launch it.
func NewThread() (*Thread, error) {
	threadLibInitOnce.Do(func() {
		if res := C.spdk_thread_lib_init(nil, 0); res != 0 {
			logger.Fatal("spdk_thread_lib_init error",
				zap.Error(eal.Errno(-res)),
			)
		}
	})

	spdkThread := C.spdk_thread_create(nil, nil)
	if spdkThread == nil {
		return nil, errors.New("spdk_thread_create error")
	}

	th := &Thread{
		c:           (*C.SpdkThread)(eal.Zmalloc("SpdkThread", C.sizeof_SpdkThread, eal.NumaSocket{})),
		RcuReadSide: &urcu.ReadSide{IsOnline: true},
	}
	th.c.spdkTh = spdkThread
	th.Thread = ealthread.New(
		cptr.Func0.C(C.SpdkThread_Run, unsafe.Pointer(th.c)),
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
	C.c_SpdkThread_Run(th.c)
}

// Post asynchronously posts a function to be executed on the SPDK thread.
func (th *Thread) Post(fn cptr.Function) {
	f, arg := cptr.Func0.CallbackOnce(fn)
	C.spdk_thread_send_msg(th.c.spdkTh, C.spdk_msg_fn(f), arg)
}
