package pingmempool

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Predefined mempool templates.
var (
	// Interest is a mempool template for generated Interests.
	Interest pktmbuf.Template

	// Data is a mempool template for generated Data headers.
	Data pktmbuf.Template

	// Payload is a mempool template for generated Data payload.
	Payload pktmbuf.Template
)

func init() {
	ndnHeaderConfig := ndni.HeaderMempool.GetConfig()

	Interest = pktmbuf.RegisterTemplate("INTEREST", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: ndnHeaderConfig.PrivSize,
		Dataroom: ndnHeaderConfig.Dataroom + ndni.Interest_TailroomMax,
	})

	Data = pktmbuf.RegisterTemplate("DATA", pktmbuf.PoolConfig{
		Capacity: 65535,
		PrivSize: ndnHeaderConfig.PrivSize,
		Dataroom: ndnHeaderConfig.Dataroom + ndni.DataGen_GetTailroom0(ndni.NAME_MAX_LENGTH),
	})

	Payload = pktmbuf.RegisterTemplate("PAYLOAD", pktmbuf.PoolConfig{
		Capacity: 1023,
		PrivSize: ndnHeaderConfig.PrivSize,
		Dataroom: ndnHeaderConfig.Dataroom + ndni.DataGen_GetTailroom1(ndni.NAME_MAX_LENGTH, 9000),
	})
}
