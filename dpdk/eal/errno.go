package eal

/*
#include "../../csrc/core/common.h"

int c_rte_errno() { return rte_errno; }
*/
import "C"
import (
	"reflect"
	"strconv"
	"syscall"
)

// Errno represents a DPDK error.
type Errno syscall.Errno

func (e Errno) Error() string {
	return strconv.Itoa(int(e)) + " " + C.GoString(C.rte_strerror(C.int(e)))
}

// MakeErrno creates Errno from non-zero number or returns nil for zero.
// errno must be a signed integer.
func MakeErrno(errno interface{}) error {
	v := reflect.ValueOf(errno).Int()
	switch {
	case v == 0:
		return nil
	case v < 0:
		return Errno(-v)
	default:
		return Errno(v)
	}
}

// GetErrno returns the current DPDK error.
func GetErrno() Errno {
	return Errno(C.c_rte_errno())
}
