package cptr

import (
	"math/rand"
	"sync"
	"unsafe"
)

var ctxMap sync.Map

// CtxPut allocates void* pointer for an arbitrary Go object.
func CtxPut(obj interface{}) unsafe.Pointer {
	var id uint32
	for {
		id = rand.Uint32()
		if _, loaded := ctxMap.LoadOrStore(id, obj); !loaded {
			break
		}
	}
	return unsafe.Pointer(uintptr(id))
}

// CtxGet returns the object associated with void* pointer.
// Panics if the object is not found.
func CtxGet(ctx unsafe.Pointer) interface{} {
	id := uint32(uintptr(ctx))
	obj, ok := ctxMap.Load(id)
	if !ok {
		panic("context is missing")
	}
	return obj
}

// CtxClear deallocates void* pointer.
func CtxClear(ctx unsafe.Pointer) {
	id := uint32(uintptr(ctx))
	ctxMap.Delete(id)
}

// CtxPop is equivalent to CtxGet followed by CtxClear.
func CtxPop(ctx unsafe.Pointer) interface{} {
	obj := CtxGet(ctx)
	CtxClear(ctx)
	return obj
}
