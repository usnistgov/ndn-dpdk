package iface_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	ndnitestenv.MakePacketHeadroom = mbuftestenv.Headroom(pktmbuf.DefaultHeadroom + ndni.LpHeaderHeadroom)

	rxl = iface.NewRxLoop(eal.NumaSocket{})
	ealthread.Launch(rxl)
	txl = iface.NewTxLoop(eal.NumaSocket{})
	ealthread.Launch(txl)

	os.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	bytesFromHex = testenv.BytesFromHex
	makePacket   = ndnitestenv.MakePacket

	rxl iface.RxLoop
	txl iface.TxLoop
)
