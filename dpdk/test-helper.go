package dpdk

// This file enables unit tests to use cgo, which isn't available in *_test.go.

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L/usr/local/lib -ldpdk -lz -lrt -lm -ldl

#include <rte_errno.h>

void setErrno(int v) { rte_errno = v; }
*/
import "C"

func setErrno(v int) {
	C.setErrno(C.int(v))
}