package fetch

/*
#include "tcpcubic.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
)

func tcpCubicFromC(c *C.TcpCubic) (ca *TcpCubic) {
	return (*TcpCubic)(unsafe.Pointer(c))
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

func (ca *TcpCubic) Increase(now dpdk.TscTime, sRtt time.Duration) {
	C.TcpCubic_Increase(ca.getPtr(), C.TscTime(now), C.double(dpdk.ToTscDuration(sRtt)))
}

func (ca *TcpCubic) Decrease(now dpdk.TscTime, sRtt time.Duration) {
	C.TcpCubic_Decrease(ca.getPtr(), C.TscTime(now), C.double(dpdk.ToTscDuration(sRtt)))
}
