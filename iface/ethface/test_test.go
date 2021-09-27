package ethface_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

func TestMain(m *testing.M) {
	if len(os.Args) >= 2 && os.Args[1] == memifbridgeArg {
		memifbridgeHelper()
		os.Exit(0)
	}

	ealtestenv.Init()

	pktmbuf.Direct.Update(pktmbuf.PoolConfig{
		Dataroom: 9000, // needed by fragmentation test case
		Capacity: 16383,
	})

	testenv.Exit(m.Run())
}

var (
	makeAR   = testenv.MakeAR
	fromJSON = testenv.FromJSON
)
