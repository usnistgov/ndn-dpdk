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
// size: either uintptr from unsafe.Sizeof(x), or signed integer from C.sizeof_*.
// align: alignment requirement, in number of cachelines, must be power of 2.
func ZmallocAligned(dbgtype string, size interface{}, align int, socket NumaSocket) unsafe.Pointer {
	typeC := C.CString(dbgtype)
	defer C.free(unsafe.Pointer(typeC))

	var sizeC C.size_t
	if sz, ok := size.(uintptr); ok {
		sizeC = C.size_t(sz)
	} else {
		sizeC = C.size_t(reflect.ValueOf(size).Int())
	}

	ptr := C.rte_zmalloc_socket(typeC, sizeC, C.unsigned(align*C.RTE_CACHE_LINE_SIZE), C.int(socket))
	if ptr == nil {
		panic(fmt.Sprintf("ZmallocAligned(%d) failed", size))
	}
	return ptr
}

// Deallocate memory from Zmalloc.
func Free(ptr interface{}) {
	C.rte_free(unsafe.Pointer(reflect.ValueOf(ptr).Pointer()))
}
