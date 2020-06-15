package eal

/*
#include "../../csrc/core/common.h"
#include <rte_errno.h>

int getErrno() { return rte_errno; }
*/
import "C"
import (
	"syscall"
)

// Errno represents a DPDK error.
type Errno syscall.Errno

// GetErrno returns the current DPDK error.
func GetErrno() Errno {
	return Errno(C.getErrno())
}

func (e Errno) Error() string {
	return C.GoString(C.rte_strerror(C.int(e)))
}
