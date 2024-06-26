package eal

/*
#include "../../csrc/core/common.h"
*/
import "C"
import (
	"unsafe"

	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
)

// Zmalloc allocates zero'ed memory on specified NumaSocket.
// Panics if out of memory.
func Zmalloc[T any, S constraints.Integer](dbgtype string, size S, socket NumaSocket) *T {
	return ZmallocAligned[T](dbgtype, size, 0, socket)
}

// ZmallocAligned allocates zero'ed memory on specified NumaSocket.
// Panics if out of memory.
// align specifies alignment requirement in cachelines (must be power of 2), or 0 to indicate no requirement.
func ZmallocAligned[T any, S constraints.Integer](dbgtype string, size S, align int, socket NumaSocket) *T {
	return (*T)(zmallocImpl(dbgtype, uintptr(size), align, socket))
}

func zmallocImpl(dbgtype string, size uintptr, align int, socket NumaSocket) unsafe.Pointer {
	var typ [32]byte
	copy(typ[:len(typ)-1], dbgtype)
	typC := (*C.char)(unsafe.Pointer(unsafe.SliceData(typ[:])))

	ptr := C.rte_zmalloc_socket(typC, C.size_t(size), C.uint(align*C.RTE_CACHE_LINE_SIZE), C.int(socket.ID()))
	if ptr == nil {
		logger.Panic(
			"zmalloc failed",
			zap.String("type", dbgtype),
			zap.Uintptr("size", size),
			socket.ZapField("socket"),
		)
	}
	return ptr
}

// Free deallocates memory from Zmalloc.
func Free[T any](ptr *T) {
	C.rte_free(unsafe.Pointer(ptr))
}
