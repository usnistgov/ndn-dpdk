package eal

/*
#include "../../csrc/core/common.h"

static int c_rte_errno() { return rte_errno; }
*/
import "C"
import (
	"strconv"
	"syscall"

	"golang.org/x/exp/constraints"
)

// Errno represents a DPDK error.
type Errno syscall.Errno

func (e Errno) Error() string {
	return strconv.Itoa(int(e)) + " " + C.GoString(C.rte_strerror(C.int(e)))
}

// MakeErrno creates Errno from non-zero number or returns nil for zero.
func MakeErrno[I constraints.Signed](v I) error {
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
