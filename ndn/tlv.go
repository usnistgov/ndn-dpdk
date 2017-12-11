package ndn

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <tlv.h>
*/
import "C"
import (
	_ "fmt"
)

const VARNUM_BUFLEN = int(C.VARNUM_BUFLEN)

func EncodeVarNum(n uint64, output []byte) uint {
	if len(output) < VARNUM_BUFLEN {
		panic("output buffer is too small")
	}

	len := C.EncodeVarNum(C.uint64_t(n), (*C.uint8_t)(&output[0]))
	return uint(len)
}

func DecodeVarNum(input []byte) (uint64, uint) {
	if len(input) < VARNUM_BUFLEN {
		panic("input buffer is too small")
	}

	var n C.uint64_t
	len := C.DecodeVarNum((*C.uint8_t)(&input[0]), &n)
	return uint64(n), uint(len)
}
