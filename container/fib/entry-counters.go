package fib

import (
	"fmt"
)

// EntryCounters contains (aggregated) FIB entry counters.
type EntryCounters struct {
	NRxInterests uint64
	NRxData      uint64
	NRxNacks     uint64
	NTxInterests uint64
}

// Add accumulates an entry's counters into cnt.
func (cnt *EntryCounters) Add(entry *Entry) {
	c := entry.ptr()
	cnt.NRxInterests += uint64(c.nRxInterests)
	cnt.NRxData += uint64(c.nRxData)
	cnt.NRxNacks += uint64(c.nRxNacks)
	cnt.NTxInterests += uint64(c.nTxInterests)
}

func (cnt EntryCounters) String() string {
	return fmt.Sprintf("%dI %dD %dN %dO", cnt.NRxInterests, cnt.NRxData, cnt.NRxNacks, cnt.NTxInterests)
}
