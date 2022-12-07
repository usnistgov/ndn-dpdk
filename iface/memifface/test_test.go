package memifface_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

func TestMain(m *testing.M) {
	memiftransport.ExecBridgeHelper()
	ealtestenv.Init()
	testenv.Exit(m.Run())
}

var (
	makeAR = testenv.MakeAR
)
