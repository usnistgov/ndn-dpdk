package diskstore_test

import (
	"os"
	"testing"

	"ndn-dpdk/core/testenv"
	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/eal/ealtestenv"
	"ndn-dpdk/ndn/ndntestenv"
	"ndn-dpdk/spdk"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	spdk.MustInit(eal.ListSlaveLCores()[0])
	spdk.InitBdevLib()

	os.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndntestenv.MakeInterest
	makeData     = ndntestenv.MakeData
)
