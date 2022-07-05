// Package tgtestenv provides facility to test the traffic generator.
package tgtestenv

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Demuxes in RxLoop.
var (
	DemuxI *iface.InputDemux
	DemuxD *iface.InputDemux
	DemuxN *iface.InputDemux
)

// Init initializes testing environment for traffic generator applications.
func Init() {
	ealtestenv.Init()
	iface.RxParseFor = ndni.ParseForApp

	rxl := iface.NewRxLoop(eal.RandomSocket())
	DemuxI, DemuxD, DemuxN = rxl.DemuxOf(ndni.PktInterest), rxl.DemuxOf(ndni.PktData), rxl.DemuxOf(ndni.PktNack)
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

// Open allocates lcores to workers, then connects RxQueues.
func Open(t testing.TB, m tgdef.Module) {
	_, require := testenv.MakeAR(t)
	if e := ealthread.AllocThread(m.Workers()...); e != nil {
		require.NoError(e)
	}
	t.Cleanup(ealthread.AllocClear)

	switch m := m.(type) {
	case tgdef.Producer:
		m.ConnectRxQueues(DemuxI)
	case tgdef.Consumer:
		m.ConnectRxQueues(DemuxD, DemuxN)
	default:
		require.FailNow("unexpected traffic generator module type")
	}
}
