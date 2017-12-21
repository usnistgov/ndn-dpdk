package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

func TestCArgs(t *testing.T) {
	assert, _ := makeAR(t)

	args := []string{"a", "", "bc", "d"}
	a := dpdk.NewCArgs(args)
	defer a.Close()

	res := verifyCArgs(a)
	assert.Equal(0, res)

	rem := a.GetRemainingArgs(1)
	assert.Equal([]string{"", "d", "bc"}, rem)
}
