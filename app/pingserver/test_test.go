package pingserver_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/ping/pingtestenv"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestMain(m *testing.M) {
	pingtestenv.Init()
	os.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	makeInterest = ndnitestenv.MakeInterest
	makeData     = ndnitestenv.MakeData
)
