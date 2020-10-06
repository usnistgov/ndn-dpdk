package cptr

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"unsafe"
)

// CArgs rearranges args so that they can be provided to C code.
type CArgs struct {
	Argc    int              // argc for C code (cast to C.int)
	Argv    unsafe.Pointer   // argv for C code (cast to **C.char)
	strMems []unsafe.Pointer // C strings
}

// NewCArgs constructs CArgs.
func NewCArgs(args []string) *CArgs {
	a := new(CArgs)
	a.Argc = len(args)

	var b *C.char
	ptrSize := unsafe.Sizeof(b)
	a.Argv = C.malloc(C.size_t(ptrSize) * C.size_t(len(args)))

	for i, arg := range args {
		argvEle := (**C.char)(unsafe.Pointer(uintptr(a.Argv) + uintptr(i)*ptrSize))
		*argvEle = C.CString(arg)
		a.strMems = append(a.strMems, unsafe.Pointer(*argvEle))
	}

	return a
}

// Close releases C memory in CArgs.
func (a *CArgs) Close() error {
	for _, strMem := range a.strMems {
		C.free(strMem)
	}
	C.free(unsafe.Pointer(a.Argv))
	return nil
}
