package ndt_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.InitEal()
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
