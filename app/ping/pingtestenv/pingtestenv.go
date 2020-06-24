package pingtestenv

import (
	"github.com/usnistgov/ndn-dpdk/container/pktqueue"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
	"github.com/usnistgov/ndn-dpdk/iface/mockface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func Init() {
	ealtestenv.InitEal()

	var faceCfg createface.Config
	faceCfg.EnableMock = true
	faceCfg.Apply()

	slaves := eal.ListSlaveLCores()

	rxl := iface.NewRxLoop(slaves[0].GetNumaSocket())
	rxl.SetLCore(slaves[0])
	rxl.Launch()
	createface.AddRxLoop(rxl)

	txl := iface.NewTxLoop(slaves[1].GetNumaSocket())
	txl.SetLCore(slaves[1])
	txl.Launch()
	createface.AddTxLoop(txl)

	SlaveLCores = slaves[2:]
}

var SlaveLCores []eal.LCore

func MakeMockFace() *mockface.MockFace {
	face, e := createface.Create(mockface.NewLocator())
	if e != nil {
		panic(e)
	}
	return face.(*mockface.MockFace)
}

func MakeRxFunc(q *pktqueue.PktQueue) func(pkts ...ndni.IL3Packet) {
	return func(pkts ...ndni.IL3Packet) {
		npkts := make([]*ndni.Packet, len(pkts))
		for i, pkt := range pkts {
			npkts[i] = pkt.GetPacket()
		}
		q.Push(npkts, eal.TscNow())
	}
}
