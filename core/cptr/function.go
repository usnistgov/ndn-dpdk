package cptr

/*
#include "../../csrc/core/common.h"

extern int go_functionGo0_once(uintptr_t ctx);
extern int go_functionGo0_reuse(uintptr_t ctx);

typedef int (*Function0)(uintptr_t ctx);
typedef int (*Function1)(void* param1, uintptr_t ctx);

static int c_invokeFunction0(Function0 f, uintptr_t ctx)
{
	return (*f)(ctx);
}

static int c_invokeFunction1(Function1 f, void* param1, uintptr_t ctx)
{
	return (*f)(param1, ctx);
}
*/
import "C"
import (
	"reflect"
	"runtime/cgo"
	"unsafe"
)

// FunctionType identifies the type of a C function.
// The zero FunctionType identifies `int f(uintptr_t ctx)`.
// FunctionType{"T1"} identifies `int f(T1* item1, uintptr_t ctx)`.
type FunctionType []string

// C wraps a C function as Function.
// f must be a C function consistent with ft.
func (ft FunctionType) C(f unsafe.Pointer, arg any) Function {
	val := reflect.ValueOf(arg)
	var ctx uintptr
	switch val.Kind() {
	case reflect.Uintptr:
		ctx = uintptr(val.Uint())
	case reflect.Ptr, reflect.UnsafePointer:
		ctx = val.Pointer()
	default:
		panic("arg must be pointer or uintptr_t")
	}
	return &functionC{ft, f, ctx}
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
func (ft FunctionType) CallbackOnce(fn Function) (f unsafe.Pointer, ctx uintptr) {
	ft.Assert(fn)
	return fn.callbackOnce()
}

// CallbackReuse returns C callback function and arg that can be invoked repeatedly.
// Use revoke() to avoid memory leak.
func (ft FunctionType) CallbackReuse(fn Function) (f unsafe.Pointer, ctx uintptr, revoke func()) {
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

	f, ctx := fn.callbackOnce()
	switch len(ft) {
	case 0:
		return int(C.c_invokeFunction0(C.Function0(f), C.uintptr_t(ctx)))
	case 1:
		return int(C.c_invokeFunction1(C.Function0(f), param[0], C.uintptr_t(ctx)))
	default:
		panic("FunctionType unimplemented")
	}
}

// ZeroFunctionType is the `int f(uintptr_t ctx)` type.
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

// Func0 is the `int f(uintptr_t ctx)` type.
var Func0 ZeroFunctionType

// Function provides a C function with void* argument.
type Function interface {
	functionType() FunctionType

	callbackOnce() (f unsafe.Pointer, ctx uintptr)

	callbackReuse() (f unsafe.Pointer, ctx uintptr, revoke func())
}

// Call wraps a Go function as Function and immediately uses it.
// post is a function that asynchronously invokes fn (this must be asynchronous).
// f must be a function with zero parameters and zero or one return values.
// Returns f's return value, or nil if f does not have a return value.
func Call(post func(fn Function), f any) any {
	done := make(chan any)
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
	ctx uintptr
}

func (fn *functionC) callbackOnce() (f unsafe.Pointer, ctx uintptr) {
	return fn.f, fn.ctx
}

func (fn *functionC) callbackReuse() (f unsafe.Pointer, ctx uintptr, revoke func()) {
	return fn.f, fn.ctx, func() {}
}

func (fn *functionC) functionType() FunctionType {
	return fn.ft
}

type functionGo0 struct {
	f func() int
}

func (fn *functionGo0) callbackOnce() (f unsafe.Pointer, arg uintptr) {
	return C.go_functionGo0_once, uintptr(cgo.NewHandle(fn.f))
}

func (fn *functionGo0) callbackReuse() (f unsafe.Pointer, arg uintptr, revoke func()) {
	ctx := cgo.NewHandle(fn.f)
	return C.go_functionGo0_reuse, uintptr(ctx), func() { ctx.Delete() }
}

func (fn *functionGo0) functionType() FunctionType {
	return Func0.FunctionType
}

//export go_functionGo0_once
func go_functionGo0_once(ctxC C.uintptr_t) C.int {
	ctx := cgo.Handle(ctxC)
	defer ctx.Delete()
	f := ctx.Value().(func() int)
	return C.int(f())
}

//export go_functionGo0_reuse
func go_functionGo0_reuse(ctxC C.uintptr_t) C.int {
	ctx := cgo.Handle(ctxC)
	f := ctx.Value().(func() int)
	return C.int(f())
}
