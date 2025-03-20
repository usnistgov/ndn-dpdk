package ethringdev

import (
	"errors"
	"fmt"
	"slices"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"go4.org/must"
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
	if cfg.RxPool == nil {
		logger.Panic("cfg.RxPool is missing")
	}
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

func (cfg PairConfig) toEthDevConfig() ethdev.Config {
	return ethdev.Config{
		RxQueues: slices.Repeat([]ethdev.RxQueueConfig{{
			Capacity: cfg.QueueCapacity,
			Socket:   cfg.Socket,
			RxPool:   cfg.RxPool,
		}}, cfg.NQueues),
		TxQueues: slices.Repeat([]ethdev.TxQueueConfig{{
			Capacity: cfg.QueueCapacity,
			Socket:   cfg.Socket,
		}}, cfg.NQueues),
	}
}

// Pair represents a pair of EthDevs connected via ring-based PMD.
type Pair struct {
	cfg   PairConfig
	rings []*ringbuffer.Ring

	PortA ethdev.EthDev
	PortB ethdev.EthDev
}

// EthDevConfig returns Config that can be used to start a port.
func (pair *Pair) EthDevConfig() ethdev.Config {
	return pair.cfg.toEthDevConfig()
}

// Close stops both ports.
func (pair *Pair) Close() error {
	errs := []error{}
	if pair.PortA != nil {
		errs = append(errs, pair.PortA.Close())
	}
	if pair.PortB != nil {
		errs = append(errs, pair.PortB.Close())
	}
	for _, r := range pair.rings {
		errs = append(errs, r.Close())
	}
	return errors.Join(errs...)
}

// NewPair creates a pair of connected EthDevs.
func NewPair(cfg PairConfig) (pair *Pair, e error) {
	cfg.applyDefaults()
	pair = &Pair{cfg: cfg}
	defer func() {
		if e != nil {
			must.Close(pair)
		}
	}()

	for range cfg.NQueues * 2 {
		ring, e := ringbuffer.New(cfg.RingCapacity, cfg.Socket,
			ringbuffer.ProducerSingle, ringbuffer.ConsumerSingle)
		if e != nil {
			return nil, fmt.Errorf("ringbuffer.New %w", e)
		}
		pair.rings = append(pair.rings, ring)
	}
	ringsAB, ringsBA := pair.rings[:cfg.NQueues], pair.rings[cfg.NQueues:]

	pair.PortA, e = New(ringsAB, ringsBA, cfg.Socket)
	if e != nil {
		return nil, fmt.Errorf("ethringdev.New %w", e)
	}
	pair.PortB, e = New(ringsBA, ringsAB, cfg.Socket)
	if e != nil {
		return nil, fmt.Errorf("ethringdev.New %w", e)
	}

	return pair, nil
}
