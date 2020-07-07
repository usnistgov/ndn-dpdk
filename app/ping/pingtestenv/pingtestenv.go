package pingtestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
)

// Init initializes testing environment for ping applications.
func Init() {
	ealtestenv.Init()
	WorkerLCores = eal.Workers[2:]

	var faceCfg createface.Config
	faceCfg.EnableSock = true
	faceCfg.Apply()

	rxl := iface.NewRxLoop(eal.Workers[0].NumaSocket())
	rxl.SetLCore(eal.Workers[0])
	DemuxI, DemuxD, DemuxN = rxl.InterestDemux(), rxl.DataDemux(), rxl.NackDemux()
	DemuxI.InitFirst()
	DemuxD.InitFirst()
	DemuxN.InitFirst()
	rxl.Launch()
	createface.AddRxLoop(rxl)

	txl := iface.NewTxLoop(eal.Workers[1].NumaSocket())
	txl.SetLCore(eal.Workers[1])
	txl.Launch()
	createface.AddTxLoop(txl)
}

// WorkerLCores is a list of unused lcores.
var WorkerLCores []eal.LCore

// Demuxes in RxLoop.
var (
	DemuxI *iface.InputDemux
	DemuxD *iface.InputDemux
	DemuxN *iface.InputDemux
)
