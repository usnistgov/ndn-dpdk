package coretest

/*
int TestSipHash();
*/
import "C"
import (
	"testing"

	_ "ndn-dpdk/core"
)

func testSipHash(t *testing.T) {
	assert, _ := makeAR(t)

	assert.EqualValues(0, C.TestSipHash())
}
