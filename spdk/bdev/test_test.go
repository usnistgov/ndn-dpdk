package bdev_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/spdk/spdkenv"
)

func TestMain(m *testing.M) {
	ealtestenv.Init()
	spdkenv.Init()
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
