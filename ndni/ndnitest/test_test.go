package ndnitest

//go:generate bash ../../mk/cgotest.sh

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	directDataroom = ndni.PacketMempool.Config().Dataroom
	os.Exit(m.Run())
}
