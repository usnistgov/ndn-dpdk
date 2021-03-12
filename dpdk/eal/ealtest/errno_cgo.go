package ealtest

/*
#include <rte_config.h>
#include <rte_errno.h>

void go_setErrno(int v) { rte_errno = v; }
*/
import "C"

func setErrno(v int) {
	C.go_setErrno(C.int(v))
}
