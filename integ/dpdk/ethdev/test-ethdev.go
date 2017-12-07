package main

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
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

	nReceived := 0
	eal.Slaves[0].RemoteLaunch(func() int {
		for {
			pkts := rxq.RxBurst(RX_BURST_SIZE)
			nReceived = nReceived + len(pkts)
			for _, pkt := range pkts {
				assert.EqualValues(1, pkt.GetDataLength())
				pkt.Close()
			}
		}
	})

	eal.Slaves[1].RemoteLaunch(func() int {
		for i := 0; i < TX_LOOPS; i++ {
			var pkts [TX_BURST_SIZE]dpdk.Mbuf
			e := mp.AllocBulk(pkts[:])
			require.NoErrorf(e, "mp.Alloc error at loop %d", i)
			for j := 0; j < TX_BURST_SIZE; j++ {
				pktBuf, e := pkts[j].Append(1)
				assert.NoError(e)
				*(*byte)(pktBuf) = byte(j)
			}
			res := txq.TxBurst(pkts[:])
			assert.EqualValuesf(TX_BURST_SIZE, res, "TxBurst incomplete at loop %d", i)
		}
		return 0
	})
	eal.Slaves[1].Wait()

	fmt.Println("nReceived=", nReceived)
	fmt.Println(port.GetStats())
	assert.True(nReceived <= TX_LOOPS*TX_BURST_SIZE)
	assert.InEpsilon(TX_LOOPS*TX_BURST_SIZE, nReceived, 0.05)
}
