package cptrtest

/*
#include <string.h>

int verifyCArgs(int argc, char** const argv) {
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
import (
	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func verifyCArgs(a *cptr.CArgs) int {
	return int(C.verifyCArgs(C.int(a.Argc), (**C.char)(a.Argv)))
}
