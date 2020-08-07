package fib

import (
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
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
			entry := replica.Get(entry.Name)
			if entry != nil {
				entry.AccCounters(&cnt, replica)
			}
		}
	})
	return
}
