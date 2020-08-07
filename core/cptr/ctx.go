package cptr

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"sync"
	"unsafe"
)

var ctxMap sync.Map

// CtxPut allocates void* pointer for an arbitrary Go object.
func CtxPut(obj interface{}) (ctx unsafe.Pointer) {
	ctx = C.malloc(1)
	ctxMap.Store(uintptr(ctx), obj)
	return ctx
}

// CtxGet returns the object associated with void* pointer.
// Panics if the object is not found.
func CtxGet(ctx unsafe.Pointer) interface{} {
	obj, ok := ctxMap.Load(uintptr(ctx))
	if !ok {
		panic("context is missing")
	}
	return obj
}

// CtxClear deallocates void* pointer.
func CtxClear(ctx unsafe.Pointer) {
	ctxMap.Delete(uintptr(ctx))
	C.free(ctx)
}

// CtxPop is equivalent to CtxGet followed by CtxClear.
func CtxPop(ctx unsafe.Pointer) interface{} {
	obj := CtxGet(ctx)
	CtxClear(ctx)
	return obj
}
