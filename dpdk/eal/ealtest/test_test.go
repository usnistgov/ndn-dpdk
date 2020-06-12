package ealtest

import (
	"os"
	"testing"

	"ndn-dpdk/core/testenv"
	"ndn-dpdk/dpdk/eal/ealtestenv"
)

// Command line arguments checked in TestEal test case.
var initEalRemainingArgs []string

func TestMain(m *testing.M) {
	initEalRemainingArgs = ealtestenv.InitEal("--", "c7f36046-faa5-46dc-9855-e93d00217b8f")
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
