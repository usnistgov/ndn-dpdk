package fetch

/*
#include "tcpcubic.h"
*/
import "C"
import (
	"ndn-dpdk/dpdk/eal"
	"time"
	"unsafe"
)

func TcpCubicFromC(ptr unsafe.Pointer) (ca *TcpCubic) {
	return (*TcpCubic)(ptr)
}

func (ca *TcpCubic) getPtr() *C.TcpCubic {
	return (*C.TcpCubic)(unsafe.Pointer(ca))
}

func (ca *TcpCubic) Init() {
	C.TcpCubic_Init(ca.getPtr())
}

func (ca *TcpCubic) GetCwnd() int {
	return int(C.TcpCubic_GetCwnd(ca.getPtr()))
}

func (ca *TcpCubic) Increase(now eal.TscTime, sRtt time.Duration) {
	C.TcpCubic_Increase(ca.getPtr(), C.TscTime(now), C.double(eal.ToTscDuration(sRtt)))
}

func (ca *TcpCubic) Decrease(now eal.TscTime) {
	C.TcpCubic_Decrease(ca.getPtr(), C.TscTime(now))
}
