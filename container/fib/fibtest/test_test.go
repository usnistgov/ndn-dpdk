package fibtest

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	os.Exit(m.Run())
}
