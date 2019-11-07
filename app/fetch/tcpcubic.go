package fetch

/*
#include "tcpcubic.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
)

func NewTcpCubicAt(c *C.TcpCubic) (ca *TcpCubic) {
	ca = (*TcpCubic)(unsafe.Pointer(c))
	ca.Init()
	return ca
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

func (ca *TcpCubic) Increase(now dpdk.TscTime, sRtt int64) {
	C.TcpCubic_Increase(ca.getPtr(), C.TscTime(now), C.TscDuration(sRtt))
}

func (ca *TcpCubic) Decrease(now dpdk.TscTime, sRtt int64) {
	C.TcpCubic_Decrease(ca.getPtr(), C.TscTime(now), C.TscDuration(sRtt))
}
