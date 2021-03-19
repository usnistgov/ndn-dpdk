package fwdptest

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	os.Exit(m.Run())
}

var (
	makeAR = testenv.MakeAR
)

func lphToken(token uint64) ndn.LpL3 {
	return ndn.LpL3{PitToken: ndn.PitTokenFromUint(token)}
}
