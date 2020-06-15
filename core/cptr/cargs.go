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

// GetRemainingArgs returns remaining argv tokens after the first nConsumed tokens have been consumed by C code.
func (a *CArgs) GetRemainingArgs(nConsumed int) []string {
	var b *C.char
	ptrSize := unsafe.Sizeof(b)

	rem := []string{}
	argv := uintptr(a.Argv)
	for i := nConsumed; i < a.Argc; i++ {
		argvEle := (**C.char)(unsafe.Pointer(uintptr(argv) + uintptr(i)*ptrSize))
		rem = append(rem, C.GoString(*argvEle))
	}

	return rem
}

// Close releases C memory in CArgs.
func (a *CArgs) Close() error {
	for _, strMem := range a.strMems {
		C.free(strMem)
	}
	C.free(unsafe.Pointer(a.Argv))
	return nil
}
