package mempool_test

import (
	"os"
	"testing"

	"ndn-dpdk/core/testenv"
	"ndn-dpdk/dpdk/eal/ealtestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
