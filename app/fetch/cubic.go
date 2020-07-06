package fetch

/*
#include "../../csrc/fetch/tcpcubic.h"
*/
import "C"
import (
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// Cubic implements the TCP CUBIC congestion avoidance algorithm.
type Cubic C.TcpCubic

func (ca *Cubic) ptr() *C.TcpCubic {
	return (*C.TcpCubic)(ca)
}

// Init initializes congestion control state.
func (ca *Cubic) Init() {
	C.TcpCubic_Init(ca.ptr())
}

// Cwnd returns current congestion window.
func (ca *Cubic) Cwnd() int {
	return int(C.TcpCubic_GetCwnd(ca.ptr()))
}

// Increase increases congestion window.
func (ca *Cubic) Increase(now eal.TscTime, sRtt time.Duration) {
	C.TcpCubic_Increase(ca.ptr(), C.TscTime(now), C.double(eal.ToTscDuration(sRtt)))
}

// Decrease deceases congestion window.
func (ca *Cubic) Decrease(now eal.TscTime) {
	C.TcpCubic_Decrease(ca.ptr(), C.TscTime(now))
}
