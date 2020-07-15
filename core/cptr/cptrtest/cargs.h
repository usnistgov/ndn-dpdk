#ifndef NDNDPDK_CPTRTEST_CARGS_H
#define NDNDPDK_CPTRTEST_CARGS_H

#include <string.h>

static inline int verifyCArgs(int argc, char** const argv) {
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

#endif // NDNDPDK_CPTRTEST_CARGS_H
