package dpdktest

import (
	"log"
	"testing"
	"time"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestEthDev(t *testing.T) {
	assert, _ := makeAR(t)

	edp := dpdktestenv.NewEthDevPair(1, 1024, 64)
	rxq, txq := edp.RxqA[0], edp.TxqB[0]
	assert.False(edp.PortA.IsDown())
	assert.False(edp.PortB.IsDown())

	const RX_BURST_SIZE = 6
	const TX_LOOPS = 100000
	const TX_BURST_SIZE = 10
	const MAX_TX_RETRY = 20
	const TX_RETRY_INTERVAL = 1 * time.Millisecond
	const RX_FINISH_WAIT = 10 * time.Millisecond

	nReceived := 0
	rxBurstSizeFreq := make(map[int]int)
	rxQuit := make(chan bool)
	dpdktestenv.Eal.Slaves[0].RemoteLaunch(func() int {
		pkts := make([]dpdk.Packet, RX_BURST_SIZE)
		for {
			burstSize := rxq.RxBurst(pkts)
			rxBurstSizeFreq[burstSize]++
			for _, pkt := range pkts[:burstSize] {
				nReceived++
				assert.Equal(1, pkt.Len(), "bad RX length at %d", nReceived)
				pkt.Close()
			}

			select {
			case <-rxQuit:
				return 0
			default:
			}
		}
	})

	txRetryFreq := make(map[int]int)
	dpdktestenv.Eal.Slaves[1].RemoteLaunch(func() int {
		for i := 0; i < TX_LOOPS; i++ {
			var pkts [TX_BURST_SIZE]dpdk.Packet
			dpdktestenv.AllocBulk(dpdktestenv.MPID_DIRECT, pkts[:])
			for j := 0; j < TX_BURST_SIZE; j++ {
				e := pkts[j].GetFirstSegment().Append([]byte{byte(j)})
				assert.NoError(e)
			}

			nSent := 0
			for nRetries := 0; nRetries < MAX_TX_RETRY; nRetries++ {
				res := txq.TxBurst(pkts[nSent:])
				nSent = nSent + int(res)
				if nSent == TX_BURST_SIZE {
					txRetryFreq[nRetries]++
					break
				}
				time.Sleep(TX_RETRY_INTERVAL)
			}
			assert.Equal(TX_BURST_SIZE, nSent, "TxBurst incomplete at loop %d", i)
		}
		return 0
	})
	dpdktestenv.Eal.Slaves[1].Wait()
	time.Sleep(RX_FINISH_WAIT)
	rxQuit <- true

	log.Println("portA.stats=", edp.PortA.GetStats())
	log.Println("portB.stats=", edp.PortB.GetStats())
	log.Println("txRetryFreq=", txRetryFreq)
	log.Println("rxBurstSizeFreq=", rxBurstSizeFreq)
	assert.True(nReceived <= TX_LOOPS*TX_BURST_SIZE)
	assert.InEpsilon(TX_LOOPS*TX_BURST_SIZE, nReceived, 0.05)
}
