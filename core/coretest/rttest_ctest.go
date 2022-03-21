package coretest

/*
#include "../../csrc/core/rttest.h"
*/
import "C"
import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

type cRttEst C.RttEst

func (rtte *cRttEst) Push(now time.Time, rtt time.Duration, withinRTT bool) {
	C.RttEst_Push((*C.RttEst)(rtte), C.TscTime_FromUnixNano(C.uint64_t(now.UnixNano())), C.TscDuration(eal.ToTscDuration(rtt)))
}

func (rtte *cRttEst) Backoff() {
	C.RttEst_Backoff((*C.RttEst)(rtte))
}

func (rtte *cRttEst) SRTT() time.Duration {
	return eal.FromTscDuration(int64(rtte.rttv.sRtt))
}

func (rtte *cRttEst) RTO() time.Duration {
	return eal.FromTscDuration(int64(rtte.rto))
}

func ctestRttEst(t *testing.T) {
	rtte := &cRttEst{}
	C.RttEst_Init((*C.RttEst)(rtte))
	RunRttEstimatorTest(t, rtte)
}
