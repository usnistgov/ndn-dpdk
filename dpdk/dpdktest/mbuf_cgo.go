package dpdktest

import "C"
import "unsafe"

func c_GoBytes(src unsafe.Pointer, count int) []byte {
	return C.GoBytes(src, C.int(count))
}
