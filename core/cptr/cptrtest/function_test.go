package cptrtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func TestFunctionC0(t *testing.T) {
	assert, _ := makeAR(t)

	intFt := cptr.FunctionType{"int"}

	assert.Equal(2424, cptr.Func0.Invoke(makeCFunction0(2423)))

	assert.Panics(func() { cptr.Func0.Invoke(makeCFunction1(intFt, 0)) })
	assert.Panics(func() { cptr.Func0.Invoke(makeCFunction0(0), ptrParam1()) })
}

func TestFunctionC1(t *testing.T) {
	assert, _ := makeAR(t)

	intFt := cptr.FunctionType{"int"}
	charFt := cptr.FunctionType{"char"}

	setParam1(8)
	assert.Equal(22208, intFt.Invoke(makeCFunction1(intFt, 2775), ptrParam1()))

	assert.Panics(func() { intFt.Invoke(makeCFunction1(intFt, 0)) })
	assert.Panics(func() { charFt.Invoke(makeCFunction0(0), ptrParam1()) })
}

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
