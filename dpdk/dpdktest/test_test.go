package dpdktest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	// TestEal test case needs these parameters.
	if eal, e := dpdk.NewEal([]string{"testprog", "-l0,2,3", "-n1", "--no-pci", "--", "X"}); e == nil {
		dpdktestenv.Eal = eal
	}

	dpdktestenv.InitEal() // panics if Eal is unavailable
	dpdktestenv.MakeDirectMp(4095, 0, 256)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
