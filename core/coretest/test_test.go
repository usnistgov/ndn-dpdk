package coretest

import (
	"os"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

var makeAR = dpdktestenv.MakeAR

func TestSipHash(t *testing.T) {
	testSipHash(t)
}
