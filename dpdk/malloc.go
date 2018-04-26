package dpdk

/*
#include <rte_config.h>
#include <rte_malloc.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"
)

// Allocate zero'ed memory on specified NumaSocket.
func Zmalloc(dbgtype string, size interface{}, socket NumaSocket) unsafe.Pointer {
	return ZmallocAligned(dbgtype, size, 0, socket)
}

// Allocate zero'ed memory on specified NumaSocket.
// Panics if out of memory.
// size: C.sizeof_*, or other signed integer type.
// align: alignment requirement, in number of cachelines, must be power of 2.
func ZmallocAligned(dbgtype string, size interface{}, align int, socket NumaSocket) unsafe.Pointer {
	cType := C.CString(dbgtype)
	defer C.free(unsafe.Pointer(cType))

	ptr := C.rte_zmalloc_socket(cType, C.size_t(reflect.ValueOf(size).Int()),
		C.unsigned(align*C.RTE_CACHE_LINE_SIZE), C.int(socket))
	if ptr == nil {
		panic(fmt.Sprintf("ZmallocAligned(%d) failed", size))
	}
	return ptr
}

// Deallocate memory from Zmalloc.
func Free(ptr interface{}) {
	C.rte_free(unsafe.Pointer(reflect.ValueOf(ptr).Pointer()))
}
