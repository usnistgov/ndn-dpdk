package cryptodev_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	directMp = mbuftestenv.DirectMempool()
	testenv.Exit(m.Run())
}

var (
	makeAR       = testenv.MakeAR
	bytesFromHex = testenv.BytesFromHex
	makePacket   = mbuftestenv.MakePacket
	directMp     *pktmbuf.Pool
)
