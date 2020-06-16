package ndntestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
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
	Name.Template = ndni.NameMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	Header.Template = ndni.HeaderMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	Guider.Template = ndni.GuiderMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})

}
