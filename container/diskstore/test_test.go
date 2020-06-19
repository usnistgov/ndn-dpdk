package diskstore_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
	"github.com/usnistgov/ndn-dpdk/spdk/spdkenv"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	spdkenv.Init(eal.ListSlaveLCores()[0])
	os.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndnitestenv.MakeInterest
	makeData     = ndnitestenv.MakeData
	closePacket  = ndnitestenv.ClosePacket
)
