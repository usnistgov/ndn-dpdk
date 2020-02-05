package pingtestenv

import (
	"ndn-dpdk/appinit"
	"ndn-dpdk/container/pktqueue"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/ndn"
)

func Init() {
	faceCfg := createface.GetDefaultConfig()
	faceCfg.EnableEth = false
	faceCfg.EnableSock = false
	faceCfg.EnableMock = true
	faceCfg.Apply()

	appinit.ProvideCreateFaceMempools()
	_, mockface.FaceMempools = ifacetestfixture.MakeMempools()

	slaves := dpdk.ListSlaveLCores()

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

var SlaveLCores []dpdk.LCore

func MakeMockFace() *mockface.MockFace {
	face, e := createface.Create(mockface.NewLocator())
	if e != nil {
		panic(e)
	}
	return face.(*mockface.MockFace)
}

type IRxQueueGettable interface {
	GetRxQueue() pktqueue.PktQueue
}

func MakeRxFunc(t IRxQueueGettable) func(pkts ...ndn.IL3Packet) {
	q := t.GetRxQueue()
	return func(pkts ...ndn.IL3Packet) {
		npkts := make([]ndn.Packet, len(pkts))
		for i, pkt := range pkts {
			npkts[i] = pkt.GetPacket()
		}
		q.Push(npkts, dpdk.TscNow())
	}
}
