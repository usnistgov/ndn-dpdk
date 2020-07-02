package fibtest

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
