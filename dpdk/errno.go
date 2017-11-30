package dpdk

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <rte_errno.h>

int getErrno() { return rte_errno; }
*/
import "C"
import "syscall"

func GetErrno() syscall.Errno {
	return syscall.Errno(C.getErrno())
}
