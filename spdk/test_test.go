package spdk_test

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/spdk"
)

func TestMain(m *testing.M) {
	eal := dpdktestenv.InitEal()
	dpdktestenv.MakeDirectMp(255, 0, 2000)
	spdk.MustInit(eal, eal.Slaves[0])
	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
