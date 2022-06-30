package fetch

/*
#include "../../csrc/fetch/logic.h"
*/
import "C"
import (
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
)

// Logic implements fetcher congestion control and scheduling logic.
type Logic C.FetchLogic

func (fl *Logic) ptr() *C.FetchLogic {
	return (*C.FetchLogic)(fl)
}

// Init initializes the logic and allocates data structures.
func (fl *Logic) Init(winCapacity int, socket eal.NumaSocket) {
	C.FetchLogic_Init(fl.ptr(), C.uint32_t(winCapacity), C.int(socket.ID()))
}

// Reset resets this to initial state.
func (fl *Logic) Reset(r segmented.SegmentRange) {
	r.SegmentRangeApplyDefaults()
	C.FetchLogic_Reset(fl.ptr(), C.uint64_t(r.SegmentBegin), C.uint64_t(r.SegmentEnd))
}

// Close deallocates data structures.
func (fl *Logic) Close() error {
	C.FetchLogic_Free(fl.ptr())
	return nil
}

// Finished determines if all segments have been fetched.
func (fl *Logic) Finished() bool {
	return fl.finishTime != 0
}

// Counters retrieves counters.
func (fl *Logic) Counters() (cnt Counters) {
	cnt.Elapsed = eal.TscNow().Sub(eal.TscTime(fl.startTime))
	if fl.Finished() {
		finished := eal.TscTime(fl.finishTime).Sub(eal.TscTime(fl.startTime))
		cnt.Finished = &finished
	}
	cnt.LastRtt = eal.FromTscDuration(int64(fl.rtte.last))
	cnt.SRtt = eal.FromTscDuration(int64(fl.rtte.rttv.sRtt))
	cnt.Rto = eal.FromTscDuration(int64(fl.rtte.rto))
	cnt.Cwnd = int(C.TcpCubic_GetCwnd(&fl.ca))
	cnt.NInFlight = uint32(fl.nInFlight)
	cnt.NTxRetx = uint64(fl.nTxRetx)
	cnt.NRxData = uint64(fl.nRxData)
	return cnt
}

// Counters contains counters of Logic.
type Counters struct {
	Elapsed   time.Duration  `json:"elapsed" gqldesc:"Duration since start fetching."`
	Finished  *time.Duration `json:"finished" gqldesc:"Duration between start and finish; null if not finished."`
	LastRtt   time.Duration  `json:"lastRtt" gqldesc:"Last RTT sample."`
	SRtt      time.Duration  `json:"sRtt" gqldesc:"Smoothed RTT."`
	Rto       time.Duration  `json:"rto" gqldesc:"RTO."`
	Cwnd      int            `json:"cwnd" gqldesc:"Congestion window."`
	NInFlight uint32         `json:"nInFlight" gqldesc:"Currently in-flight Interests."`
	NTxRetx   uint64         `json:"nTxRetx" gqldesc:"Retransmitted Interests."`
	NRxData   uint64         `json:"nRxData" gqldesc:"Data satisfying pending Interests."`
}

func (cnt Counters) String() string {
	return fmt.Sprintf("rtt=%dms srtt=%dms rto=%dms cwnd=%d %dP %dR %dD",
		cnt.LastRtt.Milliseconds(), cnt.SRtt.Milliseconds(), cnt.Rto.Milliseconds(),
		cnt.Cwnd, cnt.NInFlight, cnt.NTxRetx, cnt.NRxData)
}
