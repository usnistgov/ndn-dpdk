package fileserver_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestMain(m *testing.M) {
	tgtestenv.Init()
	ndni.PayloadMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
