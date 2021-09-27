package fetch_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

func TestMain(m *testing.M) {
	tgtestenv.Init()
	testenv.Exit(m.Run())
}

var makeAR = testenv.MakeAR
