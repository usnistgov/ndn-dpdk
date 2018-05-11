package coretest

/*
int TestSipHash();
*/
import "C"
import (
	"testing"

	_ "ndn-dpdk/core"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func testSipHash(t *testing.T) {
	assert, _ := dpdktestenv.MakeAR(t)

	assert.EqualValues(0, C.TestSipHash())
}
