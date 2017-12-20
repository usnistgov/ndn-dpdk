package dpdk

import (
	"testing"
)

func TestCArgs(t *testing.T) {
	assert, _ := makeAR(t)

	args := []string{"a", "", "bc", "d"}
	a := newCArgs(args)
	defer a.Close()

	res := testCArgs(a)
	assert.Equal(0, res)

	rem := a.GetRemainingArgs(1)
	assert.Equal([]string{"", "d", "bc"}, rem)
}
