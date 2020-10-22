package tgconsumer_test

import (
	"os"
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

func TestMain(m *testing.M) {
	tgtestenv.Init()
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
