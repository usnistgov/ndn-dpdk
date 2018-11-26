package dpdktest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	// TestEal test case needs these parameters.
	if eal, e := dpdk.NewEal([]string{"testprog", "-l", "0,2,3", "-n", "1", "--no-pci", "--", "X"}); e == nil {
		dpdktestenv.Eal = eal
	} else {
		panic(e)
	}

	dpdktestenv.MakeDirectMp(4095, 0, 256)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
