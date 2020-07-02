package pingtestenv

import (
	"github.com/usnistgov/ndn-dpdk/app/inputdemux"
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

	Demux3 = inputdemux.NewDemux3(eal.Workers[0].NumaSocket())
	Demux3.GetInterestDemux().InitFirst()
	Demux3.GetDataDemux().InitFirst()
	Demux3.GetNackDemux().InitFirst()

	rxl := iface.NewRxLoop(eal.Workers[0].NumaSocket())
	rxl.SetLCore(eal.Workers[0])
	rxl.SetCallback(inputdemux.Demux3_FaceRx, Demux3.Ptr())
	rxl.Launch()
	createface.AddRxLoop(rxl)

	txl := iface.NewTxLoop(eal.Workers[1].NumaSocket())
	txl.SetLCore(eal.Workers[1])
	txl.Launch()
	createface.AddTxLoop(txl)
}

// WorkerLCores is a list of unused lcores.
var WorkerLCores []eal.LCore

// Demux3 is the demuxer in RxLoop.
var Demux3 *inputdemux.Demux3
