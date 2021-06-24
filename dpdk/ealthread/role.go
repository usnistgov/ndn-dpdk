package ealthread

import (
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// ThreadWithRole is a thread that identifies itself with a role.
type ThreadWithRole interface {
	Thread
	ThreadRole() string
}

// AllocThread allocates lcores to threads.
// If thread already has an allocated lcore, it remains unchanged.
// If thread type implements eal.WithNumaSocket, the lcore comes from the preferred NUMA socket.
func AllocThread(threads ...ThreadWithRole) error {
	requests := make([]AllocReq, len(threads))
	for i, th := range threads {
		var req AllocReq
		if !th.LCore().Valid() {
			req.Role = th.ThreadRole()
			if thn, ok := th.(eal.WithNumaSocket); ok {
				req.Socket = thn.NumaSocket()
			}
		}
		requests[i] = req
	}

	list, e := AllocRequest(requests...)
	if e != nil {
		return e
	}

	for i, lc := range list {
		if lc.Valid() {
			threads[i].SetLCore(lc)
		}
	}
	return nil
}

// AllocLaunch allocates lcore to a thread from DefaultAllocator, and launches the thread.
func AllocLaunch(th ThreadWithRole) error {
	if e := AllocThread(th); e != nil {
		return e
	}
	Launch(th)
	return nil
}
