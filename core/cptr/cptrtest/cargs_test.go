package cptrtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func TestCArgs(t *testing.T) {
	assert, _ := makeAR(t)

	args := []string{"a", "", "bc", "d"}
	a := cptr.NewCArgs(args)
	defer a.Close()

	res := verifyCArgs(a)
	assert.Equal(0, res)

	rem := a.GetRemainingArgs(1)
	assert.Equal([]string{"", "d", "bc"}, rem)
}
