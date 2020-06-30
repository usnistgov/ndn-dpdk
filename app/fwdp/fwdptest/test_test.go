package fwdptest

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	os.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndnitestenv.MakeInterest
	makeData     = ndnitestenv.MakeData
)

func lphToken(token uint64) ndn.LpHeader {
	return ndn.LpHeader{PitToken: ndn.PitTokenFromUint(token)}
}
