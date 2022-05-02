package cryptodev_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	testenv.Exit(m.Run())
}

var (
	makeAR     = testenv.MakeAR
	makePacket = mbuftestenv.MakePacket
)
