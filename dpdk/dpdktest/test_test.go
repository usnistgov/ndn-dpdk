package dpdktest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

// Command line arguments checked in TestEal test case.
var initEalRemainingArgs []string

func TestMain(m *testing.M) {
	initEalRemainingArgs = dpdktestenv.InitEal("--", "c7f36046-faa5-46dc-9855-e93d00217b8f")
	dpdktestenv.MakeDirectMp(4095, 0, 256)
	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
