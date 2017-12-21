package main

import (
	"fmt"
	"time"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/integ"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()
	assert, require := integ.MakeAR(t)

	eal := dpdktestenv.InitEal()
	mp := dpdktestenv.MakeDirectMp(4095, 0, 256)

	ring01, e := dpdk.NewRing("RING_tx0rx1", 4, dpdk.NUMA_SOCKET_ANY, true, true)
	require.NoError(e)
	ring10, e := dpdk.NewRing("RING_tx1rx0", 1024, dpdk.NUMA_SOCKET_ANY, true, true)
	require.NoError(e)
	ringPort0, e := dpdk.NewEthDevFromRings("A", []dpdk.Ring{ring10}, []dpdk.Ring{ring01},
		dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	ringPort1, e := dpdk.NewEthDevFromRings("B", []dpdk.Ring{ring01}, []dpdk.Ring{ring10},
		dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	ringPort0 = ringPort1
	ringPort1 = ringPort0

	assert.EqualValues(2, dpdk.CountEthDevs())
	ports := dpdk.ListEthDevs()
	require.Len(ports, 2)
	port0, port1 := ports[0], ports[1]
	assert.Equal("net_ring_A", port0.GetName())
	assert.Equal("net_ring_B", port1.GetName())

	var portConf dpdk.EthDevConfig
	portConf.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: 64, Socket: dpdk.NUMA_SOCKET_ANY, Mp: mp})
	portConf.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: 64, Socket: dpdk.NUMA_SOCKET_ANY})
	rxqs0, txqs0, e := port0.Configure(portConf)
	require.NoError(e)
	require.Len(rxqs0, 1)
	require.Len(txqs0, 1)
	rxqs1, txqs1, e := port1.Configure(portConf)
	require.NoError(e)
	require.Len(rxqs1, 1)
	require.Len(txqs1, 1)
	rxq, txq := rxqs0[0], txqs1[0]

	port0.Start()
	port1.Start()

	const RX_BURST_SIZE = 6
	const TX_LOOPS = 100000
	const TX_BURST_SIZE = 10
	const MAX_TX_RETRY = 20
	const TX_RETRY_INTERVAL = 1 * time.Millisecond
	const RX_FINISH_WAIT = 10 * time.Millisecond

	nReceived := 0
	rxBurstSizeFreq := make(map[int]int)
	rxQuit := make(chan int)
	eal.Slaves[0].RemoteLaunch(func() int {
		pkts := make([]dpdk.Packet, RX_BURST_SIZE)
		for {
			burstSize := rxq.RxBurst(pkts)
			rxBurstSizeFreq[burstSize]++
			for _, pkt := range pkts[:burstSize] {
				nReceived++
				assert.EqualValuesf(1, pkt.Len(), "bad RX length at %d", nReceived)
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
	eal.Slaves[1].RemoteLaunch(func() int {
		for i := 0; i < TX_LOOPS; i++ {
			var pkts [TX_BURST_SIZE]dpdk.Packet
			e := mp.AllocPktBulk(pkts[:])
			require.NoErrorf(e, "mp.AllocPktBulk error at loop %d", i)
			for j := 0; j < TX_BURST_SIZE; j++ {
				pktBuf, e := pkts[j].GetFirstSegment().Append(1)
				assert.NoError(e)
				*(*byte)(pktBuf) = byte(j)
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
			assert.EqualValuesf(TX_BURST_SIZE, nSent, "TxBurst incomplete at loop %d", i)
		}
		return 0
	})
	eal.Slaves[1].Wait()
	time.Sleep(RX_FINISH_WAIT)
	rxQuit <- 0

	fmt.Println(port0.GetStats())
	fmt.Println(port1.GetStats())
	fmt.Println("txtxRetryFreq=", txRetryFreq)
	fmt.Println("rxBurstSizeFreq=", rxBurstSizeFreq)
	assert.True(nReceived <= TX_LOOPS*TX_BURST_SIZE)
	assert.InEpsilon(TX_LOOPS*TX_BURST_SIZE, nReceived, 0.05)
}
