package ndnitest

//go:generate bash ../../mk/cgotest.sh

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	initMempools()
	os.Exit(m.Run())
}
