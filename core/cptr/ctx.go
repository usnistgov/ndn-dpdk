package cptr

import (
	"io"
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
	n := uintptr(id)
	return unsafe.Pointer(n)
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

// CtxCloser returns an io.Closer that invokes CtxClear.
func CtxCloser(ctx unsafe.Pointer) io.Closer {
	return ctxCloser{ctx}
}

type ctxCloser struct {
	ctx unsafe.Pointer
}

func (c ctxCloser) Close() error {
	CtxClear(c.ctx)
	return nil
}
