package nameset_test

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	dpdktestenv.InitEal()

	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR
