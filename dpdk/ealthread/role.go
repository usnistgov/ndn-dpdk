package ealthread

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// ErrNoLCore indicates no lcore is available for a role.
var ErrNoLCore = errors.New("no lcore available")

// ThreadWithRole is a thread that identifies itself with a role.
type ThreadWithRole interface {
	Thread
	ThreadRole() string
}

func requestFromThread(th ThreadWithRole) (req AllocRequest) {
	if th.LCore().Valid() {
		return
	}

	req.Role = th.ThreadRole()

	if thn, ok := th.(eal.WithNumaSocket); ok {
		req.Socket = thn.NumaSocket()
	}

	return
}

// AllocThread allocates lcores to threads.
// If thread type implements eal.WithNumaSocket, the lcore comes from the preferred NUMA socket.
func (la *Allocator) AllocThread(threads ...ThreadWithRole) error {
	requests := make([]AllocRequest, len(threads))
	for i, th := range threads {
		requests[i] = requestFromThread(th)
	}

	list := la.Request(requests...)
	if list == nil {
		return ErrNoLCore
	}

	for i, lc := range list {
		if lc.Valid() {
			threads[i].SetLCore(lc)
		}
	}
	return nil
}

// AllocThread allocates lcores to threads from DefaultAllocator.
func AllocThread(threads ...ThreadWithRole) error {
	return DefaultAllocator.AllocThread(threads...)
}

// AllocLaunch allocates lcore to a thread from DefaultAllocator, and launches the thread.
func AllocLaunch(th ThreadWithRole) error {
	if e := AllocThread(th); e != nil {
		return e
	}
	Launch(th)
	return nil
}
