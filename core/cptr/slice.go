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
	ptr := unsafe.SliceData([]T(value))
	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(value))
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
	ptr := unsafe.SliceData([]T(value))
	return (*R)(unsafe.Pointer(ptr))
}

// ExpandBits expands n LSBs in mask to booleans.
func ExpandBits[T ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64](n int, mask T) []bool {
	bits := make([]bool, n)
	for i := range bits {
		bits[i] = (mask & (1 << i)) != 0
	}
	return bits
}

// MapInChunksOf invokes a callback function with sub-vectors not exceeding a maximum length.
//
//	n: maximum vector length passed to callback function.
//	vec: input vector.
//	f: function to transform input vector of [1,n] elements to same number of outputs.
func MapInChunksOf[E any, V ~[]E, T any, R ~[]T](n int, vec V, f func(vec V) R) (results R) {
	results = make(R, 0, len(vec))
	for i := 0; i < len(vec); i += n {
		results = append(results, f(vec[i:min(len(vec), i+n)])...)
	}
	return
}
