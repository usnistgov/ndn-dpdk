package cptrtest

/*
int c_arg = 0;

int c_callback0(void* arg) {
	return 1 + *(int*)arg;
}

int c_callback1(void* param1, void* arg) {
	return *(int*)param1 * (1 + *(int*)arg);
}
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func makeCFunction0(arg int) cptr.Function {
	C.c_arg = C.int(arg)
	return cptr.Func0.C(C.c_callback0, unsafe.Pointer(&C.c_arg))
}

func makeCFunction1(ft cptr.FunctionType, arg int) cptr.Function {
	C.c_arg = C.int(arg)
	return ft.C(C.c_callback1, unsafe.Pointer(&C.c_arg))
}

var param1 C.int

func setParam1(v int) {
	param1 = C.int(v)
}

func ptrParam1() unsafe.Pointer {
	return unsafe.Pointer(&param1)
}
