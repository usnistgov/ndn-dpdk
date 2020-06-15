package ethdev

import (
	"fmt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
)

// PairConfig contains configuration for Pair.
type PairConfig struct {
	NQueues       int            // number of queues on EthDev
	RingCapacity  int            // ring capacity connecting pair of EthDevs
	QueueCapacity int            // queue capacity in each EthDev
	Socket        eal.NumaSocket // where to allocate data structures
	RxPool        *pktmbuf.Pool  // mempool for packet reception
}

func (cfg *PairConfig) applyDefaults() {
	if cfg.NQueues <= 0 {
		cfg.NQueues = 1
	}
	if cfg.RingCapacity <= 0 {
		cfg.RingCapacity = 1024
	}
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = 64
	}
}

func (cfg PairConfig) toEthDevConfig() (dcfg Config) {
	dcfg.AddRxQueues(cfg.NQueues, RxQueueConfig{
		Capacity: cfg.QueueCapacity,
		Socket:   cfg.Socket,
		RxPool:   cfg.RxPool,
	})
	dcfg.AddTxQueues(cfg.NQueues, TxQueueConfig{
		Capacity: cfg.QueueCapacity,
		Socket:   cfg.Socket,
	})
	return dcfg
}

var lastPairIndex = 0

// Pair represents a pair of EthDevs connected via ring-based PMD.
type Pair struct {
	dcfg Config

	PortA EthDev
	PortB EthDev

	ringsAB []*ringbuffer.Ring
	ringsBA []*ringbuffer.Ring
}

// NewPair creates a pair of connected EthDevs.
func NewPair(cfg PairConfig) (pair *Pair) {
	cfg.applyDefaults()

	pair = new(Pair)
	if cfg.RxPool == nil {
		panic("PairConfig.RxPool is missing")
	}
	pair.dcfg = cfg.toEthDevConfig()

	lastPairIndex++
	id := lastPairIndex

	createRings := func(direction string) (rings []*ringbuffer.Ring) {
		for i := 0; i < cfg.NQueues; i++ {
			name := fmt.Sprintf("ethdevPair_%d%s%d", id, direction, i)
			ring, e := ringbuffer.New(name, cfg.RingCapacity, cfg.Socket,
				ringbuffer.ProducerSingle, ringbuffer.ConsumerSingle)
			if e != nil {
				panic(fmt.Sprintf("ringbuffer.New(%s) error %v", name, e))
			}
			rings = append(rings, ring)
		}
		return rings
	}
	pair.ringsAB = createRings("AB")
	pair.ringsBA = createRings("BA")

	createPort := func(label string, rxRings, txRings []*ringbuffer.Ring) EthDev {
		name := fmt.Sprintf("ethdevPair_%d%s", id, label)
		port, e := NewFromRings(name, rxRings, txRings, cfg.Socket)
		if e != nil {
			panic(fmt.Sprintf("NewFromRings(%s) error %v", name, e))
		}
		return port
	}
	pair.PortA = createPort("A", pair.ringsBA, pair.ringsAB)
	pair.PortB = createPort("B", pair.ringsAB, pair.ringsBA)

	return pair
}

// GetEthDevConfig returns Config that can be used to start a port.
func (pair *Pair) GetEthDevConfig() Config {
	return pair.dcfg
}

// Close stops both ports.
func (pair *Pair) Close() error {
	pair.PortA.Stop(StopDetach)
	pair.PortB.Stop(StopDetach)
	for _, r := range pair.ringsAB {
		r.Close()
	}
	for _, r := range pair.ringsBA {
		r.Close()
	}
	return nil
}
