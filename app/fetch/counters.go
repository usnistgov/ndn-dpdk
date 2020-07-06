package fetch

import (
	"fmt"
	"time"
)

// Counters contains counters of Logic.
type Counters struct {
	Time      time.Time
	LastRtt   time.Duration
	SRtt      time.Duration
	Rto       time.Duration
	Cwnd      int
	NInFlight uint32 // number of in-flight Interests
	NTxRetx   uint64 // number of retransmitted Interests
	NRxData   uint64 // number of Data satisfying pending Interests
}

// ReadCounters retrieves counters.
func (fl *Logic) ReadCounters() (cnt Counters) {
	cnt.Time = time.Now()
	rtte := fl.RttEst()
	cnt.LastRtt = rtte.LastRtt()
	cnt.SRtt = rtte.SRtt()
	cnt.Rto = rtte.Rto()
	cnt.Cwnd = fl.Cubic().Cwnd()
	cnt.NInFlight = uint32(fl.ptr().nInFlight)
	cnt.NTxRetx = uint64(fl.ptr().nTxRetx)
	cnt.NRxData = uint64(fl.ptr().nRxData)
	return cnt
}

func (cnt Counters) String() string {
	return fmt.Sprintf("rtt=%dms srtt=%dms rto=%dms cwnd=%d %dP %dR %dD",
		cnt.LastRtt.Milliseconds(), cnt.SRtt.Milliseconds(), cnt.Rto.Milliseconds(),
		cnt.Cwnd, cnt.NInFlight, cnt.NTxRetx, cnt.NRxData)
}

// ComputeGoodput returns average number of Data per second.
func (cnt Counters) ComputeGoodput(prev Counters) float64 {
	t := cnt.Time.Sub(prev.Time).Seconds()
	return float64(cnt.NRxData-prev.NRxData) / t
}
