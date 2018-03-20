package fwdptest

import (
	"os"
	"testing"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(255, ndn.SizeofPacketPriv(), 2000)
	appinit.Eal = dpdktestenv.Eal

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
