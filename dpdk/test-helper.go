package dpdk

// This file enables unit tests to use cgo, which isn't available in *_test.go.

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <rte_errno.h>
#include <string.h>

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

int testRingObjTable(void** objTable) {
	if (objTable[0] != (void*)6698 ||
			objTable[1] != (void*)3110 ||
			(void*)objTable[2] != 0) {
		return 1;
	}
	objTable[3] = (void*)4891;
	return 0;
}
*/
import "C"

func setErrno(v int) {
	C.setErrno(C.int(v))
}

func testCArgs(a *cArgs) int {
	return int(C.testCArgs(a.Argc, a.Argv))
}

func testRingObjTable(ot *RingObjTable) int {
	return int(C.testRingObjTable(ot.getPointer()))
}
