package dpdk

/*
#include "../core/common.h"
#include <rte_errno.h>

int getErrno() { return rte_errno; }
*/
import "C"
import (
	"syscall"
)

// DPDK error number.
type Errno syscall.Errno

// Get rte_errno.
func GetErrno() Errno {
	return Errno(C.getErrno())
}

func (e Errno) Error() string {
	return C.GoString(C.rte_strerror(C.int(e)))
}
