package diskstore_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	packetPool = ndni.PacketMempool.Get(eal.NumaSocket{})
	testenv.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndnitestenv.MakeInterest
	makeData     = ndnitestenv.MakeData
	packetPool   *pktmbuf.Pool
)
