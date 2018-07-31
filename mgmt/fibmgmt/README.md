# ndn-dpdk/mgmt/fibmgmt

This package implements [FIB](../../container/fib/) management.

## Fib

**Fib.Info** returns global counters.

**Fib.List** lists FIB entry names.

**Fib.Insert** inserts or replaces an entry.
If forwarding strategy is not specified, the default strategy is used.
In case the default strategy has been unloaded, this command fails.

**Fib.Erase** erases an entry.

**Fib.Find** performs an exact match lookup.

**Fib.Lpm** performs a longest prefix match lookup.

**Fib.ReadEntryCounters** reads counters of an entry.
