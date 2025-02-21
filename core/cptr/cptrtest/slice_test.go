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

func TestMapInChunksOf(t *testing.T) {
	assert, _ := makeAR(t)

	vec := make([]int, 433)
	expected := make([]int, 433)
	for i := range vec {
		vec[i] = i
		expected[i] = i + 1
	}

	invocations := map[int]int{}
	results := cptr.MapInChunksOf(100, vec, func(s []int) (r []int) {
		invocations[s[0]] = len(s)
		r = make([]int, len(s))
		for i, v := range s {
			r[i] = v + 1
		}
		return
	})

	assert.Equal(expected, results)
	assert.Equal(map[int]int{
		0:   100,
		100: 100,
		200: 100,
		300: 100,
		400: 33,
	}, invocations)
}
