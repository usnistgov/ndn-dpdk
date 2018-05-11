package mintmrtest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	dpdktestenv.InitEal()

	os.Exit(m.Run())
}

func TestMinTmr(t *testing.T) {
	testMinTmr(t)
}
