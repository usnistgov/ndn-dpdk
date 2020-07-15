#ifndef NDNDPDK_CPTRTEST_CPTR_ARRAY_H
#define NDNDPDK_CPTRTEST_CPTR_ARRAY_H

int c_arg = 0;

int c_callback0(void* arg) {
	return 1 + *(int*)arg;
}

int c_callback1(void* param1, void* arg) {
	return *(int*)param1 * (1 + *(int*)arg);
}

#endif // NDNDPDK_CPTRTEST_CPTR_ARRAY_H
