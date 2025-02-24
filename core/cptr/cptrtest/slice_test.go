package cptrtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func TestExpandBits(t *testing.T) {
	assert, _ := makeAR(t)

	mask := uint32(0b00000100_10000001)
	bits := cptr.ExpandBits(12, mask)
	assert.Len(bits, 12)
	assert.Equal([]bool{true, false, false, false, false, false, false, true, false, false, true, false}, bits)
}
