# ndn-dpdk/container/fib

This package implements the **Forwarding Information Base (FIB)**.

FIB is partitioned according to the [NDT](../ndt/).
Most FIB entries only appear in one partition, as chosen by NDT lookup.
In case a FIB entry name is shorter than NDT's prefix length, the FIB entry is duplicated across all partitions.
Go `Fib` type represents the entire FIB; C `Fib` struct represents a single partition.

The FIB implements [2-stage LPM](http://ieeexplore.ieee.org/document/6665203/) algorithm for efficient Longest Prefix Match (LPM) lookups.

## Go code

`Fib` type provides methods to execute read or update commands, sequentially executed in an RCU read-side thread that internally obtains and releases RCU read lock.
Supported commands include:

* Exact match lookup.
* Longest prefix match lookup.
* Insert or update an entry.
* Erase an entry.
* Relocate entries during NDT update (see [NdtUpdater](../ndt/ndtupdater/)).

FIB uses [fibtree](./fibtree/) package to maintain a tree of FIB entry names for computing *MD* used in 2-stage LPM algorithm and for determining affected entries during NDT update.

## C code

`FibEntry` struct represents either a real entry or a virtual entry.
A real entry has `maxDepth` set to zero, and must have at least one nexthop and reference to a strategy.
A virtual entry has `maxDepth` set to non-zero, and does not have nexthops.

`Fib` struct is a thread-safe hash table.
It combines a DPDK mempool for entry allocation, and a URCU lock-free resizable RCU hash table (lfht) for indexing.

In the hashtable, key type is name TLV-VALUE, hash value is SipHash of name, and element type is `FibEntry`.
In case both a real entry and a virtual entry exist at the same name, the virtual entry appears in the hash table, while the real entry is accessible via `realEntry` pointer.

`Fib_Lpm` function implements 2-stage LPM procedure.
It is the only function intended to be called from other packages, and always returns a real entry.
The caller is responsible for obtaining and releasing the RCU read lock.

`FibEntry` carries a sequence number that is incremented upon every insertion.
This allows a PIT entry to save a reference to a FIB entry (`PitEntry_RefreshFibEntry` function), and detect whether the reference is still valid during later retrieval (`PitEntry_FindFibEntry` function).
