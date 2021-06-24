// Package tgtestenv provides facility to test the traffic generator.
package tgtestenv

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Init initializes testing environment for traffic generator applications.
func Init() {
	ealtestenv.Init()

	rxl := iface.NewRxLoop(eal.RandomSocket())
	DemuxI, DemuxD, DemuxN = rxl.InterestDemux(), rxl.DataDemux(), rxl.NackDemux()
	DemuxI.InitFirst()
	DemuxD.InitFirst()
	DemuxN.InitFirst()
	rxl.SetLCore(eal.Workers[0])
	ealthread.Launch(rxl)

	txl := iface.NewTxLoop(eal.RandomSocket())
	txl.SetLCore(eal.Workers[1])
	ealthread.Launch(txl)

	eal.Workers = eal.Workers[2:] // rxl and txl won't be freed by allocator
}

// Demuxes in RxLoop.
var (
	DemuxI *iface.InputDemux
	DemuxD *iface.InputDemux
	DemuxN *iface.InputDemux
)
