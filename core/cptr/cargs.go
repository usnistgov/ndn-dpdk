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
	C.free(unsafe.Pointer(a.Argv))
	return nil
}

// NewCArgs constructs CArgs.
func NewCArgs(args []string) (a *CArgs) {
	ptrSize := int(unsafe.Sizeof((*C.char)(nil)))
	a = &CArgs{
		Argc: len(args),
		Argv: C.malloc(C.size_t(ptrSize) * C.size_t(len(args))),
	}

	for i, arg := range args {
		argvEle := (**C.char)(unsafe.Add(a.Argv, i*ptrSize))
		*argvEle = C.CString(arg)
		a.strMems = append(a.strMems, unsafe.Pointer(*argvEle))
	}
	return a
}
