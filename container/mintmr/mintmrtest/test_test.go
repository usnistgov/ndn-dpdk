package mintmrtest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/eal/ealtestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()

	os.Exit(m.Run())
}

func TestMinTmr(t *testing.T) {
	testMinTmr(t)
}
