package ethdev_test

import (
	"log"
	"testing"
	"time"

	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/ethdev"
	"ndn-dpdk/dpdk/pktmbuf"
	"ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

func TestEthDev(t *testing.T) {
	assert, _ := makeAR(t)
	slaves := eal.ListSlaveLCores()

	pair := ethdev.NewPair(ethdev.PairConfig{RxPool: mbuftestenv.Direct.Pool()})
	defer pair.Close()
	pair.PortA.Start(pair.GetEthDevConfig())
	pair.PortB.Start(pair.GetEthDevConfig())
	assert.False(pair.PortA.IsDown())
	assert.False(pair.PortB.IsDown())

	rxq := pair.PortA.ListRxQueues()[0]
	txq := pair.PortB.ListTxQueues()[0]

	const rxBurstSize = 6
	const txLoops = 100000
	const txBurstSize = 10
	const maxTxRetry = 20
	const txRetryInterval = 1 * time.Millisecond
	const rxFinishWait = 10 * time.Millisecond

	nReceived := 0
	rxBurstSizeFreq := make(map[int]int)
	rxQuit := make(chan bool)
	slaves[0].RemoteLaunch(func() int {
		for {
			vec := make(pktmbuf.Vector, rxBurstSize)
			burstSize := rxq.RxBurst(vec)
			rxBurstSizeFreq[burstSize]++
			for _, pkt := range vec[:burstSize] {
				if assert.NotNil(pkt) {
					nReceived++
					assert.Equal(1, pkt.Len(), "bad RX length at %d", nReceived)
				}
			}
			vec.Close()

			select {
			case <-rxQuit:
				return 0
			default:
			}
		}
	})

	txRetryFreq := make(map[int]int)
	slaves[1].RemoteLaunch(func() int {
		for i := 0; i < txLoops; i++ {
			vec := mbuftestenv.Direct.Pool().MustAlloc(txBurstSize)
			for j := 0; j < txBurstSize; j++ {
				vec[j].Append([]byte{byte(j)})
			}

			nSent := 0
			for nRetries := 0; nRetries < maxTxRetry; nRetries++ {
				res := txq.TxBurst(vec[nSent:])
				nSent = nSent + int(res)
				if nSent == txBurstSize {
					txRetryFreq[nRetries]++
					break
				}
				time.Sleep(txRetryInterval)
			}
			assert.Equal(txBurstSize, nSent, "TxBurst incomplete at loop %d", i)
		}
		return 0
	})
	slaves[1].Wait()
	time.Sleep(rxFinishWait)
	rxQuit <- true
	slaves[0].Wait()

	log.Println("portA.stats=", pair.PortA.GetStats())
	log.Println("portB.stats=", pair.PortB.GetStats())
	log.Println("txRetryFreq=", txRetryFreq)
	log.Println("rxBurstSizeFreq=", rxBurstSizeFreq)
	assert.True(nReceived <= txLoops*txBurstSize)
	assert.InEpsilon(txLoops*txBurstSize, nReceived, 0.05)
}
