package fetch

/*
#include "../../csrc/fetch/tcpcubic.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"time"
	"unsafe"
)

func TcpCubicFromC(ptr unsafe.Pointer) (ca *TcpCubic) {
	return (*TcpCubic)(ptr)
}

func (ca *TcpCubic) ptr() *C.TcpCubic {
	return (*C.TcpCubic)(unsafe.Pointer(ca))
}

func (ca *TcpCubic) Init() {
	C.TcpCubic_Init(ca.ptr())
}

func (ca *TcpCubic) GetCwnd() int {
	return int(C.TcpCubic_GetCwnd(ca.ptr()))
}

func (ca *TcpCubic) Increase(now eal.TscTime, sRtt time.Duration) {
	C.TcpCubic_Increase(ca.ptr(), C.TscTime(now), C.double(eal.ToTscDuration(sRtt)))
}

func (ca *TcpCubic) Decrease(now eal.TscTime) {
	C.TcpCubic_Decrease(ca.ptr(), C.TscTime(now))
}
