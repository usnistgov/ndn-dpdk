package ealthread

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// ErrNoLCore indicates no lcore is available for a role.
var ErrNoLCore = errors.New("no lcore available")

type threadAllocatable interface {
	Thread
	ThreadRole() string
}

// AllocThread allocates lcore to a thread.
// If th implements eal.WithNumaSocket, the lcore comes from the preferred NUMA socket.
func (la *Allocator) AllocThread(th threadAllocatable) error {
	if th.LCore().Valid() {
		return nil
	}

	var socket eal.NumaSocket
	if thn, ok := th.(eal.WithNumaSocket); ok {
		socket = thn.NumaSocket()
	}

	lc := la.Alloc(th.ThreadRole(), socket)
	if !lc.Valid() {
		return ErrNoLCore
	}
	th.SetLCore(lc)
	return nil
}

// AllocThread allocates lcore to a thread from DefaultAllocator.
func AllocThread(th threadAllocatable) error {
	return DefaultAllocator.AllocThread(th)
}

// Launch allocates lcore to a thread from DefaultAllocator, and launches the thread.
func Launch(th threadAllocatable) error {
	if e := AllocThread(th); e != nil {
		return e
	}
	return th.Launch()
}
