package cptr

import (
	"reflect"
	"unsafe"
)

/*
#include "../../csrc/core/common.h"

extern int go_runFunction(void*);

typedef int (*CallbackFunction)(void* arg);

static int c_invokeFunction(CallbackFunction f, void* arg)
{
	return (*f)(arg);
}
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

// Call wraps a Go function as Function and immediately uses it.
// post is a function that asynchronously invokes fn.
// f must be a function with zero parameters and zero or one return values.
// Returns f's return value, or nil if f does not have a return value.
func Call(post func(fn Function), f interface{}) interface{} {
	done := make(chan interface{})
	post(VoidFunction(func() {
		res := reflect.ValueOf(f).Call(nil)
		if len(res) > 0 {
			done <- res[0].Interface()
		} else {
			done <- nil
		}
	}))
	return <-done
}

// Invoke invokes the function.
func Invoke(fn Function) int {
	f, arg := fn.MakeCFunction()
	return int(C.c_invokeFunction(C.CallbackFunction(f), arg))
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
