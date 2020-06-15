package cptrtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func TestCptrArray(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Panics(func() { cptr.ParseCptrArray(1) })
	assert.Panics(func() { cptr.ParseCptrArray("x") })
	assert.Panics(func() { cptr.ParseCptrArray([]string{"x", "y"}) })

	_, count := cptr.ParseCptrArray([]cIntPtr{})
	assert.Equal(0, count)

	ptr, count := cptr.ParseCptrArray([]cIntPtr{getCIntPtr(0), getCIntPtr(1)})
	assert.Equal(2, count)
	assert.True(checkCIntPtrArray(ptr))
}
