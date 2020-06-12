package mbuftestenv

import (
	"sync"

	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/eal/ealtestenv"
	"ndn-dpdk/dpdk/pktmbuf"
)

// TestPool adds convenience functions to pktmbuf.Pool for unit testing.
type TestPool struct {
	Template pktmbuf.Template
	poolInit sync.Once
	pool     *pktmbuf.Pool
}

var (
	// Direct provides mempool for direct mbufs.
	Direct TestPool

	// Indirect provides mempool for indirect mbufs.
	Indirect TestPool
)

func init() {
	Direct.Template = pktmbuf.Direct.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})

	Indirect.Template = pktmbuf.Indirect.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
}

// Pool returns the mempool.
func (p *TestPool) Pool() *pktmbuf.Pool {
	p.poolInit.Do(func() {
		ealtestenv.InitEal()
		p.pool = p.Template.MakePool(eal.NumaSocket{})
	})
	return p.pool
}

// Alloc allocates a packet.
func (p *TestPool) Alloc() *pktmbuf.Packet {
	vec := p.Pool().MustAlloc(1)
	return vec[0]
}
