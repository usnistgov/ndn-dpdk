package pingtestenv

import (
	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/mockface"
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
