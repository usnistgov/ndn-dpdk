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

// Close releases C memory in CArgs.
func (a *CArgs) Close() error {
	for _, strMem := range a.strMems {
		C.free(strMem)
	}
	C.free(a.Argv)
	return nil
}

// NewCArgs constructs CArgs.
func NewCArgs(args []string) (a *CArgs) {
	a = &CArgs{
		Argc: len(args),
		Argv: C.calloc(C.size_t(len(args)), C.size_t(unsafe.Sizeof((*C.char)(nil)))),
	}
	argv := unsafe.Slice((**C.char)(a.Argv), len(args))

	for i, arg := range args {
		s := C.CString(arg)
		argv[i] = s
		a.strMems = append(a.strMems, unsafe.Pointer(s))
	}
	return a
}
