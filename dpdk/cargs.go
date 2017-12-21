package dpdk

/*
#include <stdlib.h>
*/
import "C"
import "unsafe"

// Provide argc and argv to C code.
type CArgs struct {
	Argc    int              // argc for C code (cast to C.int)
	Argv    unsafe.Pointer   // argv for C code (cast to **C.char)
	strMems []unsafe.Pointer // C strings
}

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

// Get remaining argv tokens after the first nConsumed tokens have been consumed by C code.
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

func (a *CArgs) Close() {
	for _, strMem := range a.strMems {
		C.free(strMem)
	}
	C.free(unsafe.Pointer(a.Argv))
}
