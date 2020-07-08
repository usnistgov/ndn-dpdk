package ethface_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
)

var ethPortCfg ethface.PortConfig

func TestMain(m *testing.M) {
	ealtestenv.Init()

	ethPortCfg.RxqFrames = 64
	ethPortCfg.TxqPkts = 64
	ethPortCfg.TxqFrames = 64

	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
