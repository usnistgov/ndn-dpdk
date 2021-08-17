package cptr

/*
#include "../../csrc/core/common.h"

extern int go_functionGo0_once(void* ctx);
extern int go_functionGo0_reuse(void* ctx);

typedef int (*Function0)(void* arg);
typedef int (*Function1)(void* param1, void* arg);

static int c_invokeFunction0(Function0 f, void* arg)
{
	return (*f)(arg);
}

static int c_invokeFunction1(Function1 f, void* param1, void* arg)
{
	return (*f)(param1, arg);
}
*/
import "C"
import (
	"reflect"
	"runtime/cgo"
	"unsafe"
)

// FunctionType identifies the type of a C function.
// The zero FunctionType identifies `int f(void* arg)`.
// FunctionType{"T1"} identifies `int f(T1* item1, void* arg)`.
type FunctionType []string

// C wraps a C function as Function.
// f must be a C function consistent with ft.
func (ft FunctionType) C(f, arg unsafe.Pointer) Function {
	return &functionC{ft, f, arg}
}

// Assert panics if fn is not created from ft.
func (ft FunctionType) Assert(fn Function) {
	actual := fn.functionType()
	if len(ft) != len(actual) {
		panic("FunctionType mismatch")
	}
	for i, t := range ft {
		if t != actual[i] {
			panic("FunctionType mismatch")
		}
	}
}

// CallbackOnce returns C callback function and arg that can be invoked only once.
func (ft FunctionType) CallbackOnce(fn Function) (f, arg unsafe.Pointer) {
	ft.Assert(fn)
	return fn.callbackOnce()
}

// CallbackReuse returns C callback function and arg that can be invoked repeatedly.
// Use revoke() to avoid memory leak.
func (ft FunctionType) CallbackReuse(fn Function) (f, arg unsafe.Pointer, revoke func()) {
	ft.Assert(fn)
	return fn.callbackReuse()
}

// Invoke invokes the function.
// fn and param must be consistent with ft.
func (ft FunctionType) Invoke(fn Function, param ...unsafe.Pointer) int {
	ft.Assert(fn)
	if len(ft) != len(param) {
		panic("FunctionType param mismatch")
	}

	f, arg := fn.callbackOnce()
	switch len(ft) {
	case 0:
		return int(C.c_invokeFunction0(C.Function0(f), arg))
	case 1:
		return int(C.c_invokeFunction1(C.Function0(f), param[0], arg))
	default:
		panic("FunctionType unimplemented")
	}
}

// ZeroFunctionType is the `int f(void* arg)` type.
type ZeroFunctionType struct {
	FunctionType
}

// Int wraps a Go function as Function.
func (ZeroFunctionType) Int(f func() int) Function {
	return &functionGo0{f}
}

// Void wraps a Go function as Function.
func (ft ZeroFunctionType) Void(f func()) Function {
	return ft.Int(func() int {
		f()
		return 0
	})
}

// Func0 is the `int f(void* arg)` type.
var Func0 ZeroFunctionType

// Function provides a C function with void* argument.
type Function interface {
	functionType() FunctionType

	callbackOnce() (f, arg unsafe.Pointer)

	callbackReuse() (f, arg unsafe.Pointer, revoke func())
}

// Call wraps a Go function as Function and immediately uses it.
// post is a function that asynchronously invokes fn (this must be asynchronous).
// f must be a function with zero parameters and zero or one return values.
// Returns f's return value, or nil if f does not have a return value.
func Call(post func(fn Function), f interface{}) interface{} {
	done := make(chan interface{})
	post(Func0.Void(func() {
		res := reflect.ValueOf(f).Call(nil)
		if len(res) > 0 {
			done <- res[0].Interface()
		} else {
			done <- nil
		}
	}))
	return <-done
}

type functionC struct {
	ft  FunctionType
	f   unsafe.Pointer
	arg unsafe.Pointer
}

func (fn *functionC) callbackOnce() (f, arg unsafe.Pointer) {
	return fn.f, fn.arg
}

func (fn *functionC) callbackReuse() (f, arg unsafe.Pointer, revoke func()) {
	return fn.f, fn.arg, func() {}
}

func (fn *functionC) functionType() FunctionType {
	return fn.ft
}

type functionGo0 struct {
	f func() int
}

func (fn *functionGo0) callbackOnce() (f, arg unsafe.Pointer) {
	return C.go_functionGo0_once, unsafe.Pointer(cgo.NewHandle(fn.f))
}

func (fn *functionGo0) callbackReuse() (f, arg unsafe.Pointer, revoke func()) {
	ctx := cgo.NewHandle(fn.f)
	return C.go_functionGo0_reuse, unsafe.Pointer(ctx), func() { ctx.Delete() }
}

func (fn *functionGo0) functionType() FunctionType {
	return Func0.FunctionType
}

//export go_functionGo0_once
func go_functionGo0_once(ctx0 unsafe.Pointer) C.int {
	ctx := cgo.Handle(ctx0)
	defer ctx.Delete()
	f := ctx.Value().(func() int)
	return C.int(f())
}

//export go_functionGo0_reuse
func go_functionGo0_reuse(ctx unsafe.Pointer) C.int {
	f := cgo.Handle(ctx).Value().(func() int)
	return C.int(f())
}
