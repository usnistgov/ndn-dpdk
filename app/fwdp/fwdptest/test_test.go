package fwdptest

import (
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	os.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndntestenv.MakeInterest
	makeData     = ndntestenv.MakeData
	getPitToken  = ndntestenv.GetPitToken
	setPitToken  = ndntestenv.SetPitToken
	copyPitToken = ndntestenv.CopyPitToken
)
