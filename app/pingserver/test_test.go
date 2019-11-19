package pingserver_test

import (
	"os"
	"testing"

	"ndn-dpdk/app/ping/pingtestenv"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(1023, ndn.SizeofPacketPriv(), 2000)
	pingtestenv.Init()
	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
