package dpdktest

/*
#include <stdbool.h>

int* getCIntPtr(int index) {
	static int arr[2] = {0xAAA1, 0xAAA2};
	return &arr[index];
}

bool checkCIntPtrArray(int** arr) {
	return *arr[0] == 0xAAA1 && *arr[1] == 0xAAA2;
}
*/
import "C"
import "unsafe"

type cIntPtr *C.int

func getCIntPtr(index int) cIntPtr {
	return C.getCIntPtr(C.int(index))
}

func checkCIntPtrArray(ptr unsafe.Pointer) bool {
	return bool(C.checkCIntPtrArray((**C.int)(ptr)))
}
