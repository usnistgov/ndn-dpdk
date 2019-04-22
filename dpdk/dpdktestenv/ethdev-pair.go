package dpdktestenv

import (
	"fmt"

	"ndn-dpdk/dpdk"
)

// Configuration for EthDevPair.
type EthDevPairConfig struct {
	NQueues       int    // number of queues on EthDev
	RingCapacity  int    // ring capacity connecting pair of EthDevs
	QueueCapacity int    // queue capacity in each EthDev
	MempoolId     string // mempool for packet reception, created via MakeMp()
}

func (cfg *EthDevPairConfig) ApplyDefaults() {
	if cfg.NQueues <= 0 {
		cfg.NQueues = 1
	}
	if cfg.RingCapacity <= 0 {
		cfg.RingCapacity = 1024
	}
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = 64
	}
	if cfg.MempoolId == "" {
		cfg.MempoolId = MPID_DIRECT
	}
}

func (cfg EthDevPairConfig) asPortConf() (portConf dpdk.EthDevConfig) {
	mp := GetMp(cfg.MempoolId)
	for i := 0; i < cfg.NQueues; i++ {
		portConf.AddRxQueue(dpdk.EthRxQueueConfig{Capacity: cfg.QueueCapacity, Socket: dpdk.NUMA_SOCKET_ANY, Mp: mp})
		portConf.AddTxQueue(dpdk.EthTxQueueConfig{Capacity: cfg.QueueCapacity, Socket: dpdk.NUMA_SOCKET_ANY})
	}
	return portConf
}

var lastEthDevPairId = 0

// A pair of EthDevs connected via ring-based PMD.
type EthDevPair struct {
	cfg EthDevPairConfig

	PortA dpdk.EthDev
	RxqA  []dpdk.EthRxQueue
	TxqA  []dpdk.EthTxQueue
	PortB dpdk.EthDev
	RxqB  []dpdk.EthRxQueue
	TxqB  []dpdk.EthTxQueue

	ringsAB []dpdk.Ring
	ringsBA []dpdk.Ring

	startedA bool
	startedB bool
}

func NewEthDevPair(cfg EthDevPairConfig) (edp *EthDevPair) {
	lastEthDevPairId++
	id := lastEthDevPairId

	edp = new(EthDevPair)
	edp.cfg = cfg
	edp.cfg.ApplyDefaults()

	edp.ringsAB = make([]dpdk.Ring, edp.cfg.NQueues)
	edp.ringsBA = make([]dpdk.Ring, edp.cfg.NQueues)
	createRings := func(label string) (rings []dpdk.Ring) {
		for i := 0; i < edp.cfg.NQueues; i++ {
			name := fmt.Sprintf("EthDevPair_%d%s%d", id, label, i)
			ring, e := dpdk.NewRing(name, edp.cfg.RingCapacity, dpdk.NUMA_SOCKET_ANY, true, true)
			if e != nil {
				panic(fmt.Sprintf("dpdk.NewRing(%s) error %v", name, e))
			}
			rings = append(rings, ring)
		}
		return rings
	}
	edp.ringsAB = createRings("AB")
	edp.ringsBA = createRings("BA")

	createPort := func(label string, rxRings []dpdk.Ring, txRings []dpdk.Ring) dpdk.EthDev {
		name := fmt.Sprintf("EthDevPair_%d%s", id, label)
		port, e := dpdk.NewEthDevFromRings(name, rxRings, txRings, dpdk.NUMA_SOCKET_ANY)
		if e != nil {
			panic(fmt.Sprintf("dpdk.NewEthDevFromRings(%s) error %v", name, e))
		}
		return port
	}
	edp.PortA = createPort("A", edp.ringsBA, edp.ringsAB)
	edp.PortB = createPort("B", edp.ringsAB, edp.ringsBA)

	return edp
}

func (edp *EthDevPair) StartPortA() {
	if edp.startedA {
		return
	}
	var e error
	edp.RxqA, edp.TxqA, e = edp.PortA.Configure(edp.cfg.asPortConf())
	if e != nil {
		panic(fmt.Sprintf("EthDev(A).Configure error %v", e))
	}
	edp.PortA.Start()
	edp.startedA = true
}

func (edp *EthDevPair) StartPortB() {
	if edp.startedB {
		return
	}
	var e error
	edp.RxqB, edp.TxqB, e = edp.PortB.Configure(edp.cfg.asPortConf())
	if e != nil {
		panic(fmt.Sprintf("EthDev(B).Configure error %v", e))
	}
	edp.PortB.Start()
	edp.startedB = true
}

func (edp *EthDevPair) Close() error {
	if edp.startedA {
		edp.PortA.Stop()
		edp.PortA.Close()
	}
	if edp.startedB {
		edp.PortB.Stop()
		edp.PortB.Close()
	}
	for _, r := range edp.ringsAB {
		r.Close()
	}
	for _, r := range edp.ringsBA {
		r.Close()
	}

	return nil
}
