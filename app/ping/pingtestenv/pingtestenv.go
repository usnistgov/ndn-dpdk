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
	ealtestenv.InitEal()
	slaves := eal.ListSlaveLCores()
	SlaveLCores = slaves[2:]

	var faceCfg createface.Config
	faceCfg.EnableSock = true
	faceCfg.Apply()

	Demux3 = inputdemux.NewDemux3(slaves[0].GetNumaSocket())
	Demux3.GetInterestDemux().InitFirst()
	Demux3.GetDataDemux().InitFirst()
	Demux3.GetNackDemux().InitFirst()

	rxl := iface.NewRxLoop(slaves[0].GetNumaSocket())
	rxl.SetLCore(slaves[0])
	rxl.SetCallback(inputdemux.Demux3_FaceRx, Demux3.GetPtr())
	rxl.Launch()
	createface.AddRxLoop(rxl)

	txl := iface.NewTxLoop(slaves[1].GetNumaSocket())
	txl.SetLCore(slaves[1])
	txl.Launch()
	createface.AddTxLoop(txl)
}

// SlaveLCores is a list of unused lcores.
var SlaveLCores []eal.LCore

// Demux3 is the demuxer in RxLoop.
var Demux3 *inputdemux.Demux3
