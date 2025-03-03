package fetchtest

/*
#include "../../../csrc/fetch/logic.h"
*/
import "C"
import (
	"math/rand/v2"
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
)

func ctestLogic(t *testing.T) {
	assert, _ := makeAR(t)

	fl := eal.Zmalloc[fetch.Logic]("FetchLogic", unsafe.Sizeof(fetch.Logic{}), eal.NumaSocket{})
	defer eal.Free(fl)
	fl.Init(64, eal.NumaSocket{})
	defer fl.Close()
	flC := (*C.FetchLogic)(unsafe.Pointer(fl))

	const finalSeg = 1999
	const lossRate = 0.05

	fl.Reset(segmented.SegmentRange{
		SegmentEnd: finalSeg + 1,
	})

	rxData := make(chan uint64)
	txCounts := map[uint64]int{}
	for !fl.Finished() {
		time.Sleep(10 * time.Microsecond)
		C.MinSched_Trigger(flC.sched)

	RX:
		for {
			select {
			case rxSegNum := <-rxData:
				C.FetchLogic_RxDataBurst(flC, &C.FetchLogicRxData{segNum: C.uint64_t(rxSegNum)}, 1, C.TscTime(eal.TscNow()))
			default:
				break RX
			}
		}

		for {
			var txSegNumC C.uint64_t
			if nTx := C.FetchLogic_TxInterestBurst(flC, &txSegNumC, 1, C.TscTime(eal.TscNow())); nTx == 0 {
				break
			}
			txSegNum := uint64(txSegNumC)

			txCounts[txSegNum]++
			if loss := rand.Float64(); loss < lossRate {
				// packet loss
			} else {
				go func() {
					time.Sleep(time.Duration(5000+rand.IntN(1000)) * time.Microsecond)
					rxData <- txSegNum
				}()
			}
		}
	}

	txCountFreq := make([]int, 10)
	for i := range uint64(finalSeg + 1) {
		txCount := txCounts[i]
		assert.Greater(txCount, 0, "%d", i)
		if txCount >= len(txCountFreq) {
			txCountFreq[len(txCountFreq)-1]++
		} else {
			txCountFreq[txCount]++
		}
	}
	t.Log(txCountFreq)
	assert.Greater(txCountFreq[1], 1700)
	assert.Less(txCountFreq[9], 20)
}
