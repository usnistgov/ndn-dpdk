package tgconsumer_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestMain(m *testing.M) {
	tgtestenv.Init()
	testenv.Exit(m.Run())
}

var (
	makeAR    = testenv.MakeAR
	nameEqual = ndntestenv.NameEqual
)
