package dpdktest

/*
#include <stdlib.h>
#include <rte_config.h>
#include <rte_errno.h>
#include <rte_mbuf.h>
*/
import "C"
import "unsafe"

type c_struct_rte_mbuf C.struct_rte_mbuf

func c_memset(dest unsafe.Pointer, ch uint8, count int) {
	C.memset(dest, C.int(ch), C.size_t(count))
}

func c_GoBytes(src unsafe.Pointer, count int) []byte {
	return C.GoBytes(src, C.int(count))
}

func c_malloc(size int) unsafe.Pointer {
	return C.malloc(C.size_t(size))
}

func c_free(ptr unsafe.Pointer) {
	C.free(ptr)
}
