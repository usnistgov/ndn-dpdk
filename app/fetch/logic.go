package fetch

/*
#include "../../csrc/fetch/logic.h"
*/
import "C"
import (
	"fmt"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// Logic implements fetcher congestion control and scheduling logic.
type Logic C.FetchLogic

func (fl *Logic) ptr() *C.FetchLogic {
	return (*C.FetchLogic)(fl)
}

// Init initializes the logic and allocates data structures.
func (fl *Logic) Init(windowCapacity int, socket eal.NumaSocket) {
	C.FetchWindow_Init(&fl.win, C.uint32_t(windowCapacity), C.int(socket.ID()))
	C.RttEst_Init(&fl.rtte)
	C.TcpCubic_Init(&fl.ca)
	C.FetchLogic_Init_(fl.ptr())
}

// Reset resets this to initial state.
func (fl *Logic) Reset() {
	*fl = Logic{win: fl.win, sched: fl.sched}
	fl.win.loSegNum, fl.win.hiSegNum = 0, 0
	C.RttEst_Init(&fl.rtte)
	C.TcpCubic_Init(&fl.ca)
	C.FetchLogic_Init_(fl.ptr())
}

// Close deallocates data structures.
func (fl *Logic) Close() error {
	C.MinSched_Close(fl.sched)
	C.FetchWindow_Free(&fl.win)
	return nil
}

// SetFinalSegNum assigns (inclusive) final segment number.
func (fl *Logic) SetFinalSegNum(segNum uint64) {
	fl.finalSegNum = C.uint64_t(segNum)
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
