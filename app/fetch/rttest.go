package fetch

/*
#include "../../csrc/fetch/rttest.h"
*/
import "C"
import (
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// RttEst represents an RTT estimator.
type RttEst C.RttEst

func (rtte *RttEst) ptr() *C.RttEst {
	return (*C.RttEst)(rtte)
}

// Init initializes RTT estimator.
func (rtte *RttEst) Init() {
	C.RttEst_Init(rtte.ptr())
}

// LastRtt returns last RTT.
func (rtte *RttEst) LastRtt() time.Duration {
	return eal.FromTscDuration(int64(rtte.last))
}

// SRtt returns smoothed RTT.
func (rtte *RttEst) SRtt() time.Duration {
	return eal.FromTscDuration(int64(rtte.sRtt))
}

// Rto returns RTO.
func (rtte *RttEst) Rto() time.Duration {
	return eal.FromTscDuration(int64(rtte.rto))
}

// Push adds an RTT sample.
func (rtte *RttEst) Push(now eal.TscTime, rtt time.Duration) {
	C.RttEst_Push(rtte.ptr(), C.TscTime(now), C.TscDuration(eal.ToTscDuration(rtt)))
}

// Backoff performs RTO backoff once.
func (rtte *RttEst) Backoff() {
	C.RttEst_Backoff(rtte.ptr())
}
