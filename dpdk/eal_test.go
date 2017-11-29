package dpdk

import (
  "testing"
  "github.com/stretchr/testify/assert"
)

func TestCArgs(t *testing.T) {
	args := []string{"a", "", "bc", "d"}
	a := newCArgs(args)
	defer a.Close()

	res := testCArgs(a)
	assert.Equal(t, res, 0, "testCArgs C function error")

	rem := a.GetRemainingArgs(1)
	assert.Equal(t, rem, []string{"", "d", "bc"})
}