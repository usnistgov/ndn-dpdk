# ndn-dpdk/container/fib

This package implements the **Forwarding Information Base (FIB)**.

FIB is partitioned according to the [NDT](../ndt/).
Most FIB entries only appear in one partition, as chosen by NDT lookup.
In case a FIB entry name is shorter than NDT's prefix length, the FIB entry is duplicated across all partitions.
Go `Fib` type represents the entire FIB; C `Fib` struct represents a single partition.

The FIB implements [2-stage LPM](http://ieeexplore.ieee.org/document/6665203/) algorithm for efficient Longest Prefix Match (LPM) lookups.

## Go code

`Fib` type provides `Insert` and `Erase` methods for updates, as well as `Find` and `Lpm` methods for exact match and longest prefix match lookups.
A `commandLoop` goroutine sequentially executes all commands in an RCU read-side thread, and internally obtains and releases RCU read lock.

`Fib` type internally maintains a tree of FIB entry names.
This tree allows computing *MD* used in 2-stage LPM algorithm.

## C code

`Fib` struct is a hash table, a customization of [Thread-Safe Hash Table (TSHT)](../tsht/).
Key type is name TLV-VALUE.
Hash value is SipHash of name.
Element type is `FibEntry` that represents either a FIB entry or a virtual entry (in 2-stage LPM).

`Fib_Lpm` function implements 2-stage LPM procedure.
It is the only function intended to be called from other packages.
The caller is responsible for obtaining and releasing the RCU read lock.
