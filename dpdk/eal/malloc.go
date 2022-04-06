package eal

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"reflect"
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
)

// Zmalloc allocates zero'ed memory on specified NumaSocket.
// Panics if out of memory.
func Zmalloc[S constraints.Integer](dbgtype string, size S, socket NumaSocket) unsafe.Pointer {
	return ZmallocAligned(dbgtype, size, 0, socket)
}

// ZmallocAligned allocates zero'ed memory on specified NumaSocket.
// Panics if out of memory.
// align specifies alignment requirement in cachelines (must be power of 2), or 0 to indicate no requirement.
func ZmallocAligned[S constraints.Integer](dbgtype string, size S, align int, socket NumaSocket) unsafe.Pointer {
	return zmallocImpl(dbgtype, uintptr(size), align, socket)
}

func zmallocImpl(dbgtype string, size uintptr, align int, socket NumaSocket) unsafe.Pointer {
	var typ [32]byte
	copy(typ[:len(typ)-1], []byte(dbgtype))

	ptr := C.rte_zmalloc_socket((*C.char)(unsafe.Pointer(&typ[0])), C.size_t(size), C.uint(align*C.RTE_CACHE_LINE_SIZE), C.int(socket.ID()))
	if ptr == nil {
		logger.Panic(
			"ZmallocAligned failed",
			zap.String("type", dbgtype),
			zap.Uintptr("size", size),
			socket.ZapField("socket"),
		)
	}
	return ptr
}

// Free deallocates memory from Zmalloc.
func Free(ptr any) {
	C.rte_free(unsafe.Pointer(reflect.ValueOf(ptr).Pointer()))
}
