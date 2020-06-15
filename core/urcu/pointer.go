package urcu

/*
#include "../../csrc/core/urcu.h"

static inline void*
c_rcu_dereference(void** p)
{
	return rcu_dereference(*p);
}

static inline void*
c_rcu_xchg_pointer(void** p, void* v)
{
	return rcu_xchg_pointer(p, v);
}
*/
import "C"
import (
	"reflect"
	"unsafe"
)

// RCU protected pointer.
type Pointer struct {
	ptr *unsafe.Pointer // Address of pointer
}

// Create RCU protected pointer from C pointer.
func NewPointer(ptr interface{}) Pointer {
	return Pointer{(*unsafe.Pointer)(unsafe.Pointer(reflect.ValueOf(ptr).Pointer()))}
}

// Dereference the pointer.
func (p Pointer) Read(rs *ReadSide) unsafe.Pointer {
	rs.Lock()
	defer rs.Unlock()
	return C.c_rcu_dereference(p.ptr)
}

// Exchange pointer values.
func (p Pointer) Xchg(value unsafe.Pointer) unsafe.Pointer {
	return C.c_rcu_xchg_pointer(p.ptr, value)
}
