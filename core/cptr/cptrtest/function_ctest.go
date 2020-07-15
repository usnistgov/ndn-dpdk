package cptrtest

/*
#include "function.h"
#include <stdlib.h>
*/
import "C"
import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func ctestFunctionC0(t *testing.T) {
	assert, _ := makeAR(t)

	intFt := cptr.FunctionType{"int"}

	assert.Equal(2424, cptr.Func0.Invoke(makeCFunction0(2423)))

	param1 := C.malloc(C.sizeof_int)
	defer C.free(param1)

	assert.Panics(func() { cptr.Func0.Invoke(makeCFunction1(intFt, 0)) })
	assert.Panics(func() { cptr.Func0.Invoke(makeCFunction0(0), param1) })
}

func ctestFunctionC1(t *testing.T) {
	assert, _ := makeAR(t)

	intFt := cptr.FunctionType{"int"}
	charFt := cptr.FunctionType{"char"}

	param1 := C.malloc(C.sizeof_int)
	defer C.free(param1)
	*(*C.int)(param1) = 8
	assert.Equal(22208, intFt.Invoke(makeCFunction1(intFt, 2775), param1))

	assert.Panics(func() { intFt.Invoke(makeCFunction1(intFt, 0)) })
	assert.Panics(func() { charFt.Invoke(makeCFunction0(0), param1) })
}

func makeCFunction0(arg int) cptr.Function {
	C.c_arg = C.int(arg)
	return cptr.Func0.C(C.c_callback0, unsafe.Pointer(&C.c_arg))
}

func makeCFunction1(ft cptr.FunctionType, arg int) cptr.Function {
	C.c_arg = C.int(arg)
	return ft.C(C.c_callback1, unsafe.Pointer(&C.c_arg))
}
