package dpdk

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <rte_errno.h>

int getErrno() { return rte_errno; }
*/
import "C"
import "syscall"

type Errno syscall.Errno

func GetErrno() Errno {
	return Errno(C.getErrno())
}

func (e Errno) Error() string {
	return C.GoString(C.rte_strerror(C.int(e)))
}
