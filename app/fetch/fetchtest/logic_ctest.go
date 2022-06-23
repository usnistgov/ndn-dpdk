package fetchtest

/*
#include "../../../csrc/fetch/logic.h"
*/
import "C"
import (
	"math/rand"
	"testing"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func ctestLogic(t *testing.T) {
	assert, _ := makeAR(t)

	fl := eal.Zmalloc[fetch.Logic]("FetchLogic", unsafe.Sizeof(fetch.Logic{}), eal.NumaSocket{})
	defer eal.Free(fl)
	fl.Init(64, eal.NumaSocket{})
	defer fl.Close()
	flC := (*C.FetchLogic)(unsafe.Pointer(fl))

	const FINAL_SEG = 1999
	const LOSS_RATE = 0.05

	fl.SetFinalSegNum(FINAL_SEG)

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
			if loss := rand.Float64(); loss < LOSS_RATE {
				// packet loss
			} else {
				go func() {
					time.Sleep(time.Duration(5000+rand.Intn(1000)) * time.Microsecond)
					rxData <- txSegNum
				}()
			}
		}
	}

	txCountFreq := make([]int, 10)
	for i := uint64(0); i <= FINAL_SEG; i++ {
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
