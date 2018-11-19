package urcu

/*
#include "urcu.h"

static inline void
c_defer_rcu(void* f, void* p)
{
	defer_rcu(f, p);
}
*/
import "C"
import (
	"runtime"
	"unsafe"
)

// RCU defer-capable thread.
type DeferThread struct{}

// Register current thread an an RCU defer-capable thread.
func NewDeferThread() *DeferThread {
	runtime.LockOSThread()
	res := C.rcu_defer_register_thread()
	if res != 0 {
		panic("rcu_defer_register_thread error")
	}
	return &DeferThread{}
}

// Unregister current thread as an RCU defer-capable thread.
func (*DeferThread) Close() error {
	C.rcu_defer_unregister_thread()
	runtime.UnlockOSThread()
	return nil
}

// Invoke f after the end of a subsequent grace period.
func (*DeferThread) Defer(f unsafe.Pointer, p unsafe.Pointer) {
	C.c_defer_rcu(f, p)
}
