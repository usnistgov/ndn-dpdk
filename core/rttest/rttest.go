// Package rttest implements an RTT estimator.
package rttest

import (
	"math"
	"time"
)

//go:generate go run ../../mk/enumgen/ -guard=NDNDPDK_CORE_RTTEST_ENUM_H -out=../../csrc/core/rttest-enum.h .

// RTT algorithm constants.
const (
	K             = 4
	AlphaDividend = 1
	AlphaDivisor  = 8
	BetaDividend  = 1
	BetaDivisor   = 4
	InitRto       = 1000
	MinRto        = 200
	MaxRto        = 60000

	_ = "enumgen::RttEst"
	_ = "enumgen+5"

	Alpha          = 1.0 * AlphaDividend / AlphaDivisor
	Beta           = 1.0 * BetaDividend / BetaDivisor
	InitRtoSeconds = InitRto / 1000.0
	MinRtoSeconds  = MinRto / 1000.0
	MaxRtoSeconds  = MaxRto / 1000.0
)

// RttEstimator is an RTT estimator.
type RttEstimator struct {
	sRtt   float64
	rttVar float64
	rto    float64
}

// Push adds a measurement.
func (rtte *RttEstimator) Push(rtt time.Duration, nPendings int) {
	rttV := rtt.Seconds()
	if rtte.sRtt == 0 {
		rtte.sRtt = rttV
		rtte.rttVar = rttV / 2
	} else {
		alpha, beta := Alpha/float64(nPendings), Beta/float64(nPendings)
		rtte.rttVar = (1-beta)*rtte.rttVar + beta*math.Abs(rtte.sRtt-rttV)
		rtte.sRtt = (1-alpha)*rtte.sRtt + alpha*rttV
	}
	rtte.updateRTO()
}

// Backoff performs exponential backoff.
func (rtte *RttEstimator) Backoff() {
	rtte.setRTO(rtte.rto * 2)
}

// Assign sets SRTT and RTTVAR values.
func (rtte *RttEstimator) Assign(sRtt, rttVar time.Duration) {
	rtte.sRtt, rtte.rttVar = sRtt.Seconds(), rttVar.Seconds()
	if rtte.sRtt == 0 {
		rtte.rto = InitRtoSeconds
	} else {
		rtte.updateRTO()
	}
}

func (rtte *RttEstimator) updateRTO() {
	rtte.setRTO(rtte.sRtt + K*rtte.rttVar)
}

func (rtte *RttEstimator) setRTO(rto float64) {
	rtte.rto = math.Max(MinRtoSeconds, math.Min(rto, MaxRtoSeconds))
}

// SRTT returns smoothed round-trip time.
func (rtte RttEstimator) SRTT() time.Duration {
	return time.Duration(rtte.sRtt * float64(time.Second))
}

// RTO returns retransmission timer.
func (rtte RttEstimator) RTO() time.Duration {
	return time.Duration(rtte.rto * float64(time.Second))
}

// New creates an RTT estimator.
func New() *RttEstimator {
	return &RttEstimator{
		rto: InitRtoSeconds,
	}
}
