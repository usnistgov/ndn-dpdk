package dpdktest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

// Command line arguments checked in TestEal test case.
var initEalRemainingArgs []string

func TestMain(m *testing.M) {
	initEalRemainingArgs = dpdk.MustInitEal([]string{"testprog", "-l", "0,2,3", "-n", "1", "--no-pci", "--", "X"})
	dpdktestenv.MakeDirectMp(4095, 0, 256)
	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
