package fibtest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	dpdktestenv.MakeDirectMp(255, 0, 2000)

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
