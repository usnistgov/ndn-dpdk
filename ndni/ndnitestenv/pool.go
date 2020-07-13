package ndnitestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// TestPool instances.
var (
	Packet   = &mbuftestenv.Direct
	Indirect = &mbuftestenv.Indirect

	Header   mbuftestenv.TestPool
	Interest mbuftestenv.TestPool
	Data     mbuftestenv.TestPool
	Payload  mbuftestenv.TestPool
)

func init() {
	Header.Template = ndni.HeaderMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	Interest.Template = ndni.InterestMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	Data.Template = ndni.DataMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	Payload.Template = ndni.PayloadMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
}
