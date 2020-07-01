package fetch

/*
#include "../../csrc/fetch/rttest.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func RttEstFromPtr(ptr unsafe.Pointer) (rtte *RttEst) {
	return (*RttEst)(ptr)
}

func (rtte *RttEst) ptr() *C.RttEst {
	return (*C.RttEst)(unsafe.Pointer(rtte))
}

func (rtte *RttEst) Init() {
	C.RttEst_Init(rtte.ptr())
}

func (rtte *RttEst) GetLastRtt() time.Duration {
	return eal.FromTscDuration(rtte.Last)
}

func (rtte *RttEst) GetSRtt() time.Duration {
	return eal.FromTscDuration(int64(rtte.SRtt))
}

func (rtte *RttEst) GetRto() time.Duration {
	return eal.FromTscDuration(int64(rtte.Rto))
}

func (rtte *RttEst) Push(now eal.TscTime, rtt time.Duration) {
	C.RttEst_Push(rtte.ptr(), C.TscTime(now), C.TscDuration(eal.ToTscDuration(rtt)))
}

func (rtte *RttEst) Backoff() {
	C.RttEst_Backoff(rtte.ptr())
}
