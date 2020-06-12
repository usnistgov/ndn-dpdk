package pingtestenv

import (
	"ndn-dpdk/container/pktqueue"
	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/eal/ealtestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/ndn"
)

func Init() {
	ealtestenv.InitEal()

	faceCfg := createface.GetDefaultConfig()
	faceCfg.EnableEth = false
	faceCfg.EnableSock = false
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

func MakeRxFunc(q *pktqueue.PktQueue) func(pkts ...ndn.IL3Packet) {
	return func(pkts ...ndn.IL3Packet) {
		npkts := make([]*ndn.Packet, len(pkts))
		for i, pkt := range pkts {
			npkts[i] = pkt.GetPacket()
		}
		q.Push(npkts, eal.TscNow())
	}
}
