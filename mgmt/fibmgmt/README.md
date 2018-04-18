# ndn-dpdk/mgmt/fibmgmt

This package implements [FIB](../../container/fib/) management.

## Fib

**Fib.Info** returns global counters.

**Fib.List** lists FIB entry names.

**Fib.Insert** inserts or replaces an entry.
Currently there is no strategy management, and all entries will use `TheStrategy`.

**Fib.Erase** erases an entry.

**Fib.Find** performs an exact match lookup.

**Fib.Lpm** performs a longest prefix match lookup.
