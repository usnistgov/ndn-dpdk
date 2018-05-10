package dpdktest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	dpdktestenv.InitEal()
	dpdktestenv.MakeDirectMp(4095, 0, 256)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
