package dpdktest

/*
#include <rte_config.h>
#include <rte_errno.h>

void setErrno(int v) { rte_errno = v; }
*/
import "C"

func setErrno(v int) {
	C.setErrno(C.int(v))
}
