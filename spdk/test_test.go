package spdk_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/spdk"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	spdk.MustInit(eal.ListSlaveLCores()[0])
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
