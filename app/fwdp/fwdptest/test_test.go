package fwdptest

import (
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	os.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndnitestenv.MakeInterest
	makeData     = ndnitestenv.MakeData
	getPitToken  = ndnitestenv.GetPitToken
	setPitToken  = ndnitestenv.SetPitToken
	copyPitToken = ndnitestenv.CopyPitToken
)
