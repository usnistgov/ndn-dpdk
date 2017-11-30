package dpdk

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCArgs(t *testing.T) {
	args := []string{"a", "", "bc", "d"}
	a := newCArgs(args)
	defer a.Close()

	res := testCArgs(a)
	assert.Equal(t, 0, res, "testCArgs C function error")

	rem := a.GetRemainingArgs(1)
	assert.Equal(t, []string{"", "d", "bc"}, rem)
}
