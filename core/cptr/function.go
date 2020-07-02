package cptr

import (
	"unsafe"
)

/*
#include "../../csrc/core/common.h"

extern int go_runFunction(void*);
*/
import "C"

// Function provides a C function with void* argument.
type Function interface {
	// MakeCFunction returns `int f(void* arg)` C function and its argument.
	// The returned function can only be invoked once.
	MakeCFunction() (f, arg unsafe.Pointer)
}

// CFunction wraps a C function as Function.
// f must have signature `int f(void* arg)`.
func CFunction(f unsafe.Pointer, arg unsafe.Pointer) Function {
	return &functionC{f, arg}
}

// IntFunction wraps a Go function as Function.
func IntFunction(f func() int) Function {
	return &functionGo{f}
}

// VoidFunction wraps a Go function as Function.
func VoidFunction(f func()) Function {
	return IntFunction(func() int {
		f()
		return 0
	})
}

type functionC struct {
	f   unsafe.Pointer
	arg unsafe.Pointer
}

func (fn *functionC) MakeCFunction() (f, arg unsafe.Pointer) {
	return fn.f, fn.arg
}

type functionGo struct {
	f func() int
}

func (fn *functionGo) MakeCFunction() (f, arg unsafe.Pointer) {
	return C.go_runFunction, CtxPut(fn.f)
}

//export go_runFunction
func go_runFunction(ctx unsafe.Pointer) C.int {
	f := CtxPop(ctx).(func() int)
	return C.int(f())
}
