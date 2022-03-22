package fib

import (
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/core/rttest"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Entry represents a FIB entry.
type Entry struct {
	fibdef.Entry
	fib *Fib
}

// Counters retrieves counters, aggregated across all replicas and lookup threads.
func (entry *Entry) Counters() (cnt fibdef.EntryCounters) {
	eal.CallMain(func() {
		for _, replica := range entry.fib.replicas {
			rEntry := replica.Get(entry.Name).Real()
			if rEntry == nil {
				continue
			}
			rEntry.AccCounters(&cnt, replica)
		}
	})
	return
}

// NexthopRtts retrieves RTT estimation of each nexthop, gathered in a lookup thread.
func (entry *Entry) NexthopRtts(th LookupThread) (m map[iface.ID]*rttest.RttEstimator) {
	replica := entry.fib.replicas[th.NumaSocket()]
	replicaPtr, dynIndex := th.GetFib()
	if replica.Ptr() != replicaPtr {
		return nil
	}

	m = map[iface.ID]*rttest.RttEstimator{}
	eal.CallMain(func() {
		rEntry := replica.Get(entry.Name).Real()
		if rEntry == nil {
			return
		}
		for i, nh := range rEntry.Read().Nexthops {
			sRtt, rttVar := rEntry.NexthopRtt(dynIndex, i)
			rtte := rttest.New()
			rtte.Assign(eal.FromTscDuration(sRtt), eal.FromTscDuration(rttVar))
			m[nh] = rtte
		}
	})
	return m
}
