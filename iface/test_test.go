package iface_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	ifacetestenv.PrepareRxlTxl()
	testenv.Exit(m.Run())
}

var (
	makeAR = testenv.MakeAR
)
