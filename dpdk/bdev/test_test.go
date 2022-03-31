package bdev_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	pktmbuf.Direct.Update(pktmbuf.PoolConfig{Dataroom: 5000}) // needed for TestFile
	testenv.Exit(m.Run())
}

var (
	makeAR     = testenv.MakeAR
	makePacket = mbuftestenv.MakePacket
)
