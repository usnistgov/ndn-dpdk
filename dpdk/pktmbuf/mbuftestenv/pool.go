package mbuftestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

func init() {
	pktmbuf.Direct.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	pktmbuf.Indirect.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
}

// DirectMempool returns a mempool created from DIRECT template.
func DirectMempool() *pktmbuf.Pool {
	return pktmbuf.Direct.Get(eal.NumaSocket{})
}

// IndirectMempool returns a mempool created from INDIRECT template.
func IndirectMempool() *pktmbuf.Pool {
	return pktmbuf.Indirect.Get(eal.NumaSocket{})
}
