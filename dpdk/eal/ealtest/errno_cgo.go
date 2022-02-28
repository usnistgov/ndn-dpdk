package ealtest

/*
#include <rte_config.h>
#include <rte_errno.h>

static void c_setErrno(int v) { rte_errno = v; }
*/
import "C"

func setErrno(v int) {
	C.c_setErrno(C.int(v))
}
