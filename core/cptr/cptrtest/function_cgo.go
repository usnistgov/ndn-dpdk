package cptrtest

/*
typedef int (*CallbackFunction)(void* arg);

int invokeFunction(CallbackFunction f, void* arg) {
	return f(arg);
}

int callbackCArg = 0;

int callbackC(void* arg) {
	return 1 + *(int*)arg;
}
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func invokeFunction(fn cptr.Function) int {
	f, arg := fn.MakeCFunction()
	return int(C.invokeFunction(C.CallbackFunction(f), arg))
}

func makeCFunction(arg int) cptr.Function {
	C.callbackCArg = C.int(arg)
	return cptr.CFunction(C.callbackC, unsafe.Pointer(&C.callbackCArg))
}
