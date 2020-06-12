package pingserver_test

import (
	"os"
	"testing"

	"ndn-dpdk/app/ping/pingtestenv"
	"ndn-dpdk/core/testenv"
)

func TestMain(m *testing.M) {
	pingtestenv.Init()
	os.Exit(m.Run())
}

var makeAR = testenv.MakeAR
