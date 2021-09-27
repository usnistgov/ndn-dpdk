package ndnitest

//go:generate bash ../../mk/cgotest.sh

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	directDataroom = ndni.PacketMempool.Config().Dataroom
	testenv.Exit(m.Run())
}
