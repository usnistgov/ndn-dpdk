package rttest_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/coretest"
	"github.com/usnistgov/ndn-dpdk/core/rttest"
)

type testableRttEstimator struct {
	*rttest.RttEstimator
}

func (rtte *testableRttEstimator) Push(now time.Time, rtt time.Duration, withinRTT bool) {
	if !withinRTT {
		rtte.RttEstimator.Push(rtt, 1)
	}
}

func TestRttEstimator(t *testing.T) {
	rtte := &testableRttEstimator{
		RttEstimator: rttest.New(),
	}
	coretest.RunRttEstimatorTest(t, rtte)
}
