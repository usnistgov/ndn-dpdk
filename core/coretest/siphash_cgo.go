package coretest

/*
int TestSipHash();
*/
import "C"
import (
	"testing"

	"ndn-dpdk/core/testenv"
)

func testSipHash(t *testing.T) {
	assert, _ := testenv.MakeAR(t)

	assert.EqualValues(0, C.TestSipHash())
}
