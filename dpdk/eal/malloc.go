package eal

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"reflect"
	"unsafe"

	"go.uber.org/zap"
)

// Zmalloc allocates zero'ed memory on specified NumaSocket.
// Panics if out of memory.
func Zmalloc(dbgtype string, size any, socket NumaSocket) unsafe.Pointer {
	return ZmallocAligned(dbgtype, size, 0, socket)
}

// ZmallocAligned allocates zero'ed memory on specified NumaSocket.
// Panics if out of memory.
// size can be either uintptr or a signed integer type.
// align specifies alignment requirement in cachelines (must be power of 2), or 0 to indicate no requirement.
func ZmallocAligned(dbgtype string, size any, align int, socket NumaSocket) unsafe.Pointer {
	typeC := C.CString(dbgtype)
	defer C.free(unsafe.Pointer(typeC))

	var sizeC C.size_t
	if sz, ok := size.(uintptr); ok {
		sizeC = C.size_t(sz)
	} else {
		sizeC = C.size_t(reflect.ValueOf(size).Int())
	}

	ptr := C.rte_zmalloc_socket(typeC, sizeC, C.uint(align*C.RTE_CACHE_LINE_SIZE), C.int(socket.ID()))
	if ptr == nil {
		logger.Panic(
			"ZmallocAligned failed",
			zap.String("type", dbgtype),
			zap.Uintptr("size", uintptr(sizeC)),
			socket.ZapField("socket"),
		)
	}
	return ptr
}

// Free deallocates memory from Zmalloc.
func Free(ptr any) {
	C.rte_free(unsafe.Pointer(reflect.ValueOf(ptr).Pointer()))
}
