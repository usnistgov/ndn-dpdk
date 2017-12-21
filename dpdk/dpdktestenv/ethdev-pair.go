package dpdktestenv

import (
	"fmt"

	"ndn-dpdk/dpdk"
)

var nEthDevPairs = 0

type EthDevPair struct {
	PortA dpdk.EthDev
	RxqA  []dpdk.EthRxQueue
	TxqA  []dpdk.EthTxQueue
	PortB dpdk.EthDev
	RxqB  []dpdk.EthRxQueue
	TxqB  []dpdk.EthTxQueue

	ringsAB []dpdk.Ring
	ringsBA []dpdk.Ring
}

func NewEthDevPair(nQueues int, ringCapacity int, queueCapacity uint) (edp EthDevPair) {
	if !DirectMp.IsValid() {
		panic("NewEthDevPair requires initialized DirectMp")
	}

	var e error
	edp.ringsAB = make([]dpdk.Ring, nQueues)
	edp.ringsBA = make([]dpdk.Ring, nQueues)
	createRings := func(label string, rings []dpdk.Ring) {
		for i := range rings {
			name := fmt.Sprintf("EthDevPair_%d_%s_%d", nEthDevPairs, label, i)
			rings[i], e = dpdk.NewRing(name, ringCapacity, dpdk.NUMA_SOCKET_ANY, true, true)
			if e != nil {
				panic(fmt.Sprintf("dpdk.NewRing(%s) error %v", name, e))
			}
		}
	}
	createRings("AB", edp.ringsAB)
	createRings("BA", edp.ringsBA)

	createPort := func(label string, rxRings []dpdk.Ring, txRings []dpdk.Ring) dpdk.EthDev {
		name := fmt.Sprintf("EthDevPair_%s", label)
		port, e := dpdk.NewEthDevFromRings(name, rxRings, txRings, dpdk.NUMA_SOCKET_ANY)
		if e != nil {
			panic(fmt.Sprintf("dpdk.NewEthDevFromRings(%s) error %v", name, e))
		}
		return port
	}
	edp.PortA = createPort("A", edp.ringsBA, edp.ringsAB)
	edp.PortB = createPort("B", edp.ringsAB, edp.ringsBA)

	var portConf dpdk.EthDevConfig
	portConf.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: queueCapacity, Socket: dpdk.NUMA_SOCKET_ANY, Mp: DirectMp})
	portConf.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: queueCapacity, Socket: dpdk.NUMA_SOCKET_ANY})

	edp.RxqA, edp.TxqA, e = edp.PortA.Configure(portConf)
	if e != nil {
		panic(fmt.Sprintf("EthDev(A).Configure error %v", e))
	}
	edp.RxqB, edp.TxqB, e = edp.PortB.Configure(portConf)
	if e != nil {
		panic(fmt.Sprintf("EthDev(A).Configure error %v", e))
	}

	edp.PortA.Start()
	edp.PortB.Start()

	nEthDevPairs++
	return edp
}
