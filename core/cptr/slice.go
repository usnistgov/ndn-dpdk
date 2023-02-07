// Package cptr handles C void* pointers.
package cptr

import (
	"bytes"
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

// GetString interprets []byte as nil-terminated string.
func GetString[T ~uint8 | ~int8, A ~[]T](value A) string {
	b := AsByteSlice(value)
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return string(b)
	}
	return string(b[:i])
}

// FirstPtr returns a pointer to the first element of a slice, or nil if the slice is empty.
//
// T must be a pointer or unsafe.Pointer.
// *R should be equivalent to T.
func FirstPtr[R, T any, A ~[]T](value A) *R {
	if len(value) == 0 {
		return nil
	}
	_ = [1]byte{}[unsafe.Sizeof(value[0])-unsafe.Sizeof(unsafe.Pointer(nil))] // sizeof(T)==sizeof(void*)
	return (*R)(unsafe.Pointer(unsafe.SliceData([]T(value))))
}
