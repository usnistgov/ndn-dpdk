package fetch_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
	"unsafe"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/dpdk/eal"
)

func TestLogic(t *testing.T) {
	assert, _ := makeAR(t)

	flPtr := eal.Zmalloc("FetchLogic", unsafe.Sizeof(fetch.Logic{}), eal.NumaSocket{})
	defer eal.Free(flPtr)
	fl := fetch.LogicFromPtr(flPtr)
	fl.Init(64, eal.NumaSocket{})
	defer fl.Close()

	const FINAL_SEG = 1999
	const LOSS_RATE = 0.05

	fl.SetFinalSegNum(FINAL_SEG)

	rxData := make(chan uint64)
	txCounts := make(map[uint64]int)
	for !fl.Finished() {
		time.Sleep(10 * time.Microsecond)
		fl.TriggerRtoSched()

	RX:
		for {
			select {
			case rxSegNum := <-rxData:
				fl.RxData(rxSegNum, false)
			default:
				break RX
			}
		}

		for {
			needTx, txSegNum := fl.TxInterest()
			if !needTx {
				break
			}

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
	fmt.Println(txCountFreq)
	assert.Greater(txCountFreq[1], 1700)
	assert.Less(txCountFreq[9], 20)
}
