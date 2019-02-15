package diskstore_test

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
	"ndn-dpdk/spdk"
)

func TestMain(m *testing.M) {
	dpdktestenv.InitEal()
	spdk.MustInit(dpdk.ListSlaveLCores()[0])
	spdk.InitBdevLib()

	dpdktestenv.MakeDirectMp(255, ndn.SizeofPacketPriv(), 2000)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
