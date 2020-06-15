package ndtupdater

import (
	"time"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/ndt"
)

type NdtUpdater struct {
	Ndt      *ndt.Ndt
	Fib      *fib.Fib
	SleepFor time.Duration // wait duration for processing dispatched packets
}

func (nu *NdtUpdater) Update(index uint64, value uint8) (nRelocated int, e error) {
	oldValue := nu.Ndt.ReadElement(index)
	if oldValue == value {
		return 0, nil
	}

	e = nu.Fib.Relocate(index, oldValue, value, func(n int) error {
		nu.Ndt.Update(index, value)
		nRelocated = n
		if nRelocated > 0 {
			time.Sleep(nu.SleepFor)
		}
		return nil
	})
	return nRelocated, e
}
