package ndn

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include "tlv.h"
*/
import "C"
import (
	"errors"
)

const VARNUM_BUFLEN = int(C.VARNUM_BUFLEN)

func EncodeVarNum(n uint64, output []byte) uint {
	if len(output) < VARNUM_BUFLEN {
		panic("output buffer is too small")
	}

	len := C.EncodeVarNum(C.uint64_t(n), (*C.uint8_t)(&output[0]))
	return uint(len)
}

func DecodeVarNum(input []byte) (uint64, uint, error) {
	if len(input) < 1 {
		return 0, 0, errors.New("cannot decode from empty input")
	}
	var n C.uint64_t
	var length C.size_t
	res := C.DecodeVarNum((*C.uint8_t)(&input[0]), C.size_t(len(input)), &n, &length)
	if res != C.NdnError_OK {
		return 0, 0, NdnError(res)
	}
	return uint64(n), uint(length), nil
}
