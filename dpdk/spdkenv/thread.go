package spdkenv

/*
#include "../../csrc/dpdk/spdk-thread.h"

// workaround gopls "compiler(InvalidCall)" false positive
static int c_SpdkThread_Run(SpdkThread* th) { return SpdkThread_Run(th); }

static int c_spdk_thread_send_msg(const struct spdk_thread* th, spdk_msg_fn fn, uintptr_t ctx)
{
	return spdk_thread_send_msg(th, fn, (void*)ctx);
}
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

var initThreadLibOnce sync.Once

func initThreadLib() {
	if res := C.spdk_thread_lib_init(nil, 0); res != 0 {
		logger.Fatal("spdk_thread_lib_init error", zap.Error(eal.MakeErrno(res)))
	}
}

// Thread represents an SPDK thread.
type Thread struct {
	ealthread.ThreadWithCtrl
	name        string
	c           *C.SpdkThread
	RcuReadSide *urcu.ReadSide
}

var (
	_ eal.PollThread               = (*Thread)(nil)
	_ ealthread.ThreadWithRole     = (*Thread)(nil)
	_ ealthread.ThreadWithLoadStat = (*Thread)(nil)
)

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

	done := make(chan struct{})
	go func() {
		C.SpdkThread_Exit(th.c)
		close(done)
	}()
	<-done

	C.spdk_thread_destroy(th.c.spdkTh)
	eal.Free(th.c)
	logger.Info("SPDK thread closed",
		zap.String("name", th.name),
		zap.Uintptr("th", uintptr(unsafe.Pointer(th.c))),
	)
	return nil
}

func (th *Thread) main() {
	C.c_SpdkThread_Run(th.c)
}

// Post asynchronously posts a function to be run on the SPDK thread.
func (th *Thread) Post(fn cptr.Function) {
	f, ctx := cptr.Func0.CallbackOnce(fn)
	res := C.c_spdk_thread_send_msg(th.c.spdkTh, C.spdk_msg_fn(f), C.uintptr_t(ctx))
	if res != 0 {
		logger.Panic("spdk_thread_send_msg error", zap.Error(eal.MakeErrno(res)))
	}
}

// NewThread creates an SPDK thread.
// The caller needs to assigned it a DPDK lcore and launch it.
func NewThread() (*Thread, error) {
	initThreadLibOnce.Do(initThreadLib)

	name := eal.AllocObjectID("spdkenv.Thread")
	nameC := C.CString(name)
	defer C.free(unsafe.Pointer(nameC))
	spdkThread := C.spdk_thread_create(nameC, nil)
	if spdkThread == nil {
		return nil, errors.New("spdk_thread_create error")
	}

	th := &Thread{
		name:        name,
		c:           (*C.SpdkThread)(eal.Zmalloc("SpdkThread", C.sizeof_SpdkThread, eal.NumaSocket{})),
		RcuReadSide: &urcu.ReadSide{IsOnline: true},
	}
	th.c.spdkTh = spdkThread
	th.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.SpdkThread_Run, unsafe.Pointer(th.c)),
		unsafe.Pointer(&th.c.ctrl),
	)
	logger.Info("SPDK thread created",
		zap.String("name", name),
		zap.Uintptr("th", uintptr(unsafe.Pointer(th.c))),
		zap.Uintptr("spdk-thread", uintptr(unsafe.Pointer(th.c.spdkTh))),
	)
	return th, nil
}
