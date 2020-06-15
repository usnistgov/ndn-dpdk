package fib

/*
#include "../../csrc/fib/entry.h"
*/
import "C"
import (
	"fmt"
)

// Counters on FIB entry.
type EntryCounters struct {
	NRxInterests uint64
	NRxData      uint64
	NRxNacks     uint64
	NTxInterests uint64
}

// Add an entry's counters into cnt.
func (cnt *EntryCounters) Add(entry *Entry) {
	cnt.NRxInterests += uint64(entry.c.nRxInterests)
	cnt.NRxData += uint64(entry.c.nRxData)
	cnt.NRxNacks += uint64(entry.c.nRxNacks)
	cnt.NTxInterests += uint64(entry.c.nTxInterests)
}

func (cnt EntryCounters) String() string {
	return fmt.Sprintf("%dI %dD %dN %dO", cnt.NRxInterests, cnt.NRxData, cnt.NRxNacks,
		cnt.NTxInterests)
}
