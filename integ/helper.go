package integ

import (
	"fmt"
	"os"

	"ndn-dpdk/dpdk/dpdktestenv"
)

type Testing struct {
	hasError bool
}

func (t *Testing) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	t.hasError = true
}

func (t *Testing) FailNow() {
	os.Exit(1)
}

func (t *Testing) Close() error {
	if t.hasError {
		os.Exit(1)
	} else {
		os.Exit(0)
	}
	return nil
}

var MakeAR = dpdktestenv.MakeAR
