package ifacetestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

// Rxl and Txl created by PrepareRxlTxl.
var (
	Rxl iface.RxLoop
	Txl iface.TxLoop
)

// PrepareRxlTxl starts one RxLoop and one TxLoop.
// Packets received by the RxLoop are initially dropped.
// It also ensures ndnitestenv.MakePacket creates packets with sufficient headroom to use with iface.
func PrepareRxlTxl() {
	ndnitestenv.MakePacketHeadroom = mbuftestenv.Headroom(pktmbuf.DefaultHeadroom + ndni.LpHeaderHeadroom)

	Rxl = iface.NewRxLoop(eal.NumaSocket{})
	ealthread.AllocLaunch(Rxl)
	Txl = iface.NewTxLoop(eal.NumaSocket{})
	ealthread.AllocLaunch(Txl)
}
