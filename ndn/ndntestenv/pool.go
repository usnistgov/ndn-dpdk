package ndntestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// TestPool instances.
var (
	Packet   = &mbuftestenv.Direct
	Indirect = &mbuftestenv.Indirect
	Name     mbuftestenv.TestPool
	Header   mbuftestenv.TestPool
	Guider   mbuftestenv.TestPool
)

func init() {
	Name.Template = ndn.NameMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	Header.Template = ndn.HeaderMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	Guider.Template = ndn.GuiderMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})

}
