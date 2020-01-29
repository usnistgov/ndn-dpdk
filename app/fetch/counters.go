package fetch

import (
	"fmt"
	"time"
)

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

func (fl *Logic) ReadCounters() (cnt Counters) {
	cnt.Time = time.Now()
	cnt.LastRtt = fl.Rtte.GetLastRtt()
	cnt.SRtt = fl.Rtte.GetSRtt()
	cnt.Rto = fl.Rtte.GetRto()
	cnt.Cwnd = fl.Ca.GetCwnd()
	cnt.NInFlight = fl.NInFlight
	cnt.NTxRetx = fl.NTxRetx
	cnt.NRxData = fl.NRxData
	return cnt
}

func (cnt Counters) String() string {
	return fmt.Sprintf("rtt=%dms srtt=%dms rto=%dms cwnd=%d %dP %dR %dD",
		cnt.LastRtt.Milliseconds(), cnt.SRtt.Milliseconds(), cnt.Rto.Milliseconds(),
		cnt.Cwnd, cnt.NInFlight, cnt.NTxRetx, cnt.NRxData)
}

// Compute goodput i.e. number of Data per second.
func (cnt Counters) ComputeGoodput(prev Counters) float64 {
	t := cnt.Time.Sub(prev.Time).Seconds()
	return float64(cnt.NRxData-prev.NRxData) / t
}
