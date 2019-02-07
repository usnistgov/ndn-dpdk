package spdk

import (
	"math/rand"
	"sync"
	"unsafe"
)

var ctxMap sync.Map

// Allocate 'void* ctx' for an arbitrary object.
func ctxPut(obj interface{}) (ctx unsafe.Pointer) {
	var id uint32
	for {
		id = rand.Uint32()
		if _, loaded := ctxMap.LoadOrStore(id, obj); !loaded {
			break
		}
	}
	return unsafe.Pointer(uintptr(id))
}

// Obtain the object associated with 'void* ctx'.
func ctxGet(ctx unsafe.Pointer) (obj interface{}) {
	id := uint32(uintptr(ctx))
	obj, ok := ctxMap.Load(id)
	if !ok {
		panic("context is missing")
	}
	return obj
}

// Deallocate 'void* ctx'.
func ctxClear(ctx unsafe.Pointer) {
	id := uint32(uintptr(ctx))
	ctxMap.Delete(id)
}

// Obtain the object associated with 'void* ctx', and deallocate 'void* ctx'.
func ctxPop(ctx unsafe.Pointer) (obj interface{}) {
	obj = ctxGet(ctx)
	ctxClear(ctx)
	return obj
}
