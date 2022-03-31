package cptr

import "unsafe"

// AsByteSlice converts *[n]C.uint8_t or *[n]C.char or []C.uint8_t or []C.char to []byte.
func AsByteSlice[T ~uint8 | ~int8](value []T) (b []byte) {
	if len(value) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(&value[0])), len(value))
}
