package cptrtest

/*
#include "cargs.h"
*/
import "C"
import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func ctestCArgs(t *testing.T) {
	assert, _ := makeAR(t)

	args := []string{"a", "", "bc", "d"}
	a := cptr.NewCArgs(args)
	defer a.Close()

	assert.Zero(C.verifyCArgs(C.int(a.Argc), (**C.char)(a.Argv)))

	rem := a.RemainingArgs(1)
	assert.Equal([]string{"", "d", "bc"}, rem)
}
