package fetch

/*
#include "rttest.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"ndn-dpdk/dpdk"
)

func rttEstFromC(c *C.RttEst) (rtte *RttEst) {
	return (*RttEst)(unsafe.Pointer(c))
}

func (rtte *RttEst) getPtr() *C.RttEst {
	return (*C.RttEst)(unsafe.Pointer(rtte))
}

func (rtte *RttEst) Init() {
	C.RttEst_Init(rtte.getPtr())
}

func (rtte *RttEst) GetSRtt() time.Duration {
	return dpdk.FromTscDuration(int64(rtte.SRtt))
}

func (rtte *RttEst) GetRto() time.Duration {
	return dpdk.FromTscDuration(int64(rtte.Rto))
}

func (rtte *RttEst) Push(now dpdk.TscTime, rtt time.Duration) {
	C.RttEst_Push(rtte.getPtr(), C.TscTime(now), C.TscDuration(dpdk.ToTscDuration(rtt)))
}

func (rtte *RttEst) Backoff() {
	C.RttEst_Backoff(rtte.getPtr())
}
