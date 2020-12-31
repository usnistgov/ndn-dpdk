package ndnitestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func init() {
	ndni.HeaderMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	ndni.InterestMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	ndni.DataMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	ndni.PayloadMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
}

// MakeMempools returns mempools for packet modification.
func MakeMempools() *ndni.Mempools {
	var mp ndni.Mempools
	mp.Assign(eal.NumaSocket{})
	return &mp
}
