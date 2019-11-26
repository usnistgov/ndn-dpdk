package fetch

import (
	"fmt"
	"math"
	"time"
)

type Counters struct {
	Time      time.Time
	Rto       time.Duration
	Cwnd      int
	NInFlight uint32  // number of in-flight Interests
	NTxRetx   uint64  // number of retransmitted Interests
	NRxData   uint64  // number of Data satisfying pending Interests
	Goodput   float64 // number of Data per second
}

func (fl *Logic) ReadCounters(prev Counters) (cnt Counters) {
	cnt.Time = time.Now()
	cnt.Rto = fl.Rtte.GetRto()
	cnt.Cwnd = fl.Ca.GetCwnd()
	cnt.NInFlight = fl.NInFlight
	cnt.NTxRetx = fl.NTxRetx
	cnt.NRxData = fl.NRxData
	cnt.Goodput = math.NaN()
	if !prev.Time.IsZero() {
		t := cnt.Time.Sub(prev.Time).Seconds()
		cnt.Goodput = float64(cnt.NRxData-prev.NRxData) / t
	}
	return cnt
}

func (cnt Counters) String() string {
	return fmt.Sprintf("rto=%dms cwnd=%d %dP %dR %dD %0.0fD/s",
		cnt.Rto.Milliseconds(), cnt.Cwnd,
		cnt.NInFlight, cnt.NTxRetx, cnt.NRxData, cnt.Goodput)
}
