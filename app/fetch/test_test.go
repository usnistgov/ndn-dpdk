package fetch_test

import (
	"os"
	"testing"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/ndn"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(1023, ndn.SizeofPacketPriv(), 2000)

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

	slaveLCores = slaves[2:]

	os.Exit(m.Run())
}

var slaveLCores []dpdk.LCore

var makeAR = dpdktestenv.MakeAR

func makeMockFace() *mockface.MockFace {
	face, e := createface.Create(mockface.NewLocator())
	if e != nil {
		panic(e)
	}
	return face.(*mockface.MockFace)
}
