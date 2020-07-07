package cptrtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func TestFunction(t *testing.T) {
	assert, _ := makeAR(t)

	n := 0
	assert.Equal(0, invokeFunction(cptr.VoidFunction(func() {
		n = 8572
	})))
	assert.Equal(8572, n)

	assert.Equal(66, invokeFunction(cptr.IntFunction(func() int {
		n = 3961
		return 66
	})))
	assert.Equal(3961, n)

	assert.Equal("OK", cptr.Call(
		func(fn cptr.Function) { go invokeFunction(fn) },
		func() string { return "OK" }))

	assert.Equal(2424, invokeFunction(makeCFunction(2423)))
}
