// Package cptr handles C void* pointers.
package cptr

import (
	"unsafe"

	_ "github.com/ianlancetaylor/cgosymbolizer"
)

// AsByteSlice converts []C.uint8_t or []C.char to []byte.
func AsByteSlice[T ~uint8 | ~int8, A ~[]T](value A) (b []byte) {
	if len(value) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&value[0])), len(value))
}

// FirstPtr returns a pointer to the first element of a slice, or nil if the slice is empty.
//
// T must be a pointer or unsafe.Pointer.
// *R should be equivalent to T.
func FirstPtr[R any, T any, A ~[]T](value A) *R {
	if len(value) == 0 {
		return nil
	}
	_ = [1]byte{}[unsafe.Sizeof(value[0])-unsafe.Sizeof(unsafe.Pointer(nil))] // sizeof(T)==sizeof(void*)
	return (*R)(unsafe.Pointer(&value[0]))
}
