package spdk_test

import (
	"os"
	"testing"

	"ndn-dpdk/core/testenv"
	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/eal/ealtestenv"
	"ndn-dpdk/spdk"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	spdk.MustInit(eal.ListSlaveLCores()[0])
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
