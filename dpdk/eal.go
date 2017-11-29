package dpdk

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L/usr/local/lib -ldpdk -lz -lrt -lm -ldl -lpcap

#include <rte_config.h>
#include <rte_common.h>
#include <rte_eal.h>
#include <stdlib.h> // free()
*/
import "C"
import "unsafe"

// Provide argc and argv to C code
type cArgs struct {
	Argc C.int // argc for C code
	Argv **C.char // argv for C code
}

func newCArgs(args []string) *cArgs {
	a := new(cArgs)
	a.Argc = C.int(len(args))

	var b *C.char
	ptrSize := unsafe.Sizeof(b)
	argv := C.malloc(C.size_t(ptrSize) * C.size_t(len(args)))
	a.Argv = (**C.char)(argv)

	for i, arg := range args {
		argvEle := (**C.char)(unsafe.Pointer(uintptr(argv) + uintptr(i) * ptrSize))
		*argvEle = C.CString(arg)
	}

	return a
}

// Get remaining argv token after the first nConsumed tokens have been consumed by C code
func (a *cArgs) GetRemainingArgs(nConsumed int) []string {
	var b *C.char
	ptrSize := unsafe.Sizeof(b)

	rem := []string{}
	argv := uintptr(unsafe.Pointer(a.Argv))
	for i := nConsumed; i < int(a.Argc); i++ {
		argvEle := (**C.char)(unsafe.Pointer(uintptr(argv) + uintptr(i) * ptrSize))
		rem = append(rem, C.GoString(*argvEle))
	}

	return rem
}

func (a *cArgs) Close() {
	var b *C.char
	ptrSize := unsafe.Sizeof(b)
	argv := uintptr(unsafe.Pointer(a.Argv))
  for i := 0; i < int(a.Argc); i++ {
		argvEle := (**C.char)(unsafe.Pointer(argv + uintptr(i) * ptrSize))
		C.free(unsafe.Pointer(*argvEle))
	}

  C.free(unsafe.Pointer(a.Argv))
}