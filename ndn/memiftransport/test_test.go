package memiftransport_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

var (
	makeAR = testenv.MakeAR
)

func TestMain(m *testing.M) {
	memiftransport.ExecBridgeHelper()
	testenv.Exit(m.Run())
}
