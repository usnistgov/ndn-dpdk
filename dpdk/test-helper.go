package dpdk

// This file enables unit tests to use cgo, which isn't available in *_test.go.

/*
#include <string.h>
#include <stdlib.h>
#include <rte_config.h>
#include <rte_errno.h>
#include <rte_mbuf.h>

void setErrno(int v) { rte_errno = v; }

int testCArgs(int argc, char** const argv) {
	if (argc != 4)
		return 2;
	if (0 != strcmp(argv[0], "a") ||
			0 != strcmp(argv[1], "") ||
			0 != strcmp(argv[2], "bc") ||
			0 != strcmp(argv[3], "d")) {
		return 3;
	}
	argv[0][0] = '.';
	argv[0] = NULL;
	char* arg2 = argv[2];
	argv[2] = argv[3];
	argv[3] = arg2;
	return 0;
}
*/
import "C"
import "unsafe"

func setErrno(v int) {
	C.setErrno(C.int(v))
}

func testCArgs(a *cArgs) int {
	return int(C.testCArgs(a.Argc, a.Argv))
}

type c_struct_rte_mbuf C.struct_rte_mbuf

func c_memset(dest unsafe.Pointer, ch uint8, count uint) {
	C.memset(dest, C.int(ch), C.size_t(count))
}

func c_GoBytes(src unsafe.Pointer, count uint) []byte {
	return C.GoBytes(src, C.int(count))
}

func c_malloc(size uint) unsafe.Pointer {
	return C.malloc(C.size_t(size))
}

func c_free(ptr unsafe.Pointer) {
	C.free(ptr)
}
