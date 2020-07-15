package cptrtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func TestFunctionGo0(t *testing.T) {
	assert, _ := makeAR(t)

	n := 0
	assert.Equal(0, cptr.Func0.Invoke(cptr.Func0.Void(func() {
		n = 8572
	})))
	assert.Equal(8572, n)

	assert.Equal(66, cptr.Func0.Invoke(cptr.Func0.Int(func() int {
		n = 3961
		return 66
	})))
	assert.Equal(3961, n)

	assert.Equal("OK", cptr.Call(
		func(fn cptr.Function) { go cptr.Func0.Invoke(fn) },
		func() string { return "OK" }))
}
