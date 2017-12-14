package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
	"time"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()
	assert := assert.New(t)
	require := require.New(t)

	eal, e := dpdk.NewEal([]string{"testprog", "-l0,1,2", "--no-pci", "--vdev=net_ring0"})
	require.NoError(e)

	mp, e := dpdk.NewPktmbufPool("MP", 4095, 0, 0, 256, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	defer mp.Close()

	assert.EqualValues(1, dpdk.CountEthDevs())
	ports := dpdk.ListEthDevs()
	require.Len(ports, 1)
	port := ports[0]
	assert.NotEmpty(port.GetName())

	var portConf dpdk.EthDevConfig
	portConf.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: 64, Socket: dpdk.NUMA_SOCKET_ANY, Mp: mp})
	portConf.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: 64, Socket: dpdk.NUMA_SOCKET_ANY})
	rxqs, txqs, e := port.Configure(portConf)
	require.NoError(e)
	require.Len(rxqs, 1)
	require.Len(txqs, 1)
	rxq, txq := rxqs[0], txqs[0]

	port.Start()

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

	fmt.Println(port.GetStats())
	fmt.Println("txtxRetryFreq=", txRetryFreq)
	fmt.Println("rxBurstSizeFreq=", rxBurstSizeFreq)
	assert.True(nReceived <= TX_LOOPS*TX_BURST_SIZE)
	assert.InEpsilon(TX_LOOPS*TX_BURST_SIZE, nReceived, 0.05)
}
