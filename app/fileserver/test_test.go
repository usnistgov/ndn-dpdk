package fileserver_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestMain(m *testing.M) {
	tgtestenv.Init()
	ndni.PayloadMempool.Update(pktmbuf.PoolConfig{
		Capacity: 4095,
	})
	testenv.Exit(m.Run())
}

var (
	makeAR    = testenv.MakeAR
	randBytes = testenv.RandBytes
)
