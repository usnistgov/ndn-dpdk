package ealthread

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// lCoreProvider provides information about LCores.
// Mock of this interface allows unit testing of LCoreAllocator.
type lCoreProvider interface {
	// Workers returns a list of worker lcores.
	Workers() []eal.LCore

	// IsBusy determines whether an lcore is busy.
	IsBusy(lc eal.LCore) bool

	// NumaSocketOf returns the NUMA socket where the lcore is located.
	NumaSocketOf(lc eal.LCore) eal.NumaSocket
}

type ealLCoreProvider struct{}

func (ealLCoreProvider) Workers() []eal.LCore {
	return eal.ListSlaveLCores()
}

func (ealLCoreProvider) IsBusy(lc eal.LCore) bool {
	return lc.IsBusy()
}

func (ealLCoreProvider) NumaSocketOf(lc eal.LCore) eal.NumaSocket {
	return lc.NumaSocket()
}
