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

// Add adds an entry's counters into cnt.
func (cnt *EntryCounters) Add(entry *Entry) {
	c := (*CEntry)(entry)
	cnt.NRxInterests += uint64(c.NRxInterests)
	cnt.NRxData += uint64(c.NRxData)
	cnt.NRxNacks += uint64(c.NRxNacks)
	cnt.NTxInterests += uint64(c.NTxInterests)
}

func (cnt EntryCounters) String() string {
	return fmt.Sprintf("%dI %dD %dN %dO", cnt.NRxInterests, cnt.NRxData, cnt.NRxNacks,
		cnt.NTxInterests)
}
