package coretest

/*
int TestSipHash();
*/
import "C"
import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

func ctestSipHash(t *testing.T) {
	assert, _ := testenv.MakeAR(t)
	assert.EqualValues(0, C.TestSipHash())
}
