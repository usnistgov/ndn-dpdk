# ndn-dpdk/container/fib

This package implements the **Forwarding Information Base (FIB)**.

The FIB is partitioned according to the [NDT](../ndt).
Most FIB entries only appear in one partition, as chosen by NDT lookup.
In case a FIB entry name is shorter than the NDT's prefix length, the FIB entry will be replicated across all partitions.
The `Fib` type in Go represents the entire FIB, while the `Fib` C struct represents a single partition.

The FIB implements a [2-stage LPM](https://doi.org/10.1109/ANCS.2013.6665203) algorithm for efficient **Longest Prefix Match (LPM)** lookups.

## Go Code

The `Fib` type provides methods to execute read or update commands, sequentially executed in an RCU read-side thread that internally acquires and releases the RCU read lock.
Supported commands include:

* Exact match lookup.
* Longest prefix match lookup.
* Inserting or updating an entry.
* Erasing an entry.
* Relocating entries during an NDT update (see [NdtUpdater](../ndt/ndtupdater)).

The FIB uses the [fibtree](./fibtree) package to maintain a tree of FIB entry names for computing *MD* used in the 2-stage LPM algorithm and for determining the affected entries during an NDT update.

## C Code

The `FibEntry` struct represents either a *real entry* or a *virtual entry*.
A real entry has `maxDepth` set to zero, and must have at least one nexthop and a reference to a strategy.
Conversely, a virtual entry has `maxDepth` set to a non-zero value and does not have any nexthops.

The `Fib` struct is a thread-safe hash table.
It combines a DPDK mempool for entry allocation with a URCU lock-free resizable RCU hash table (lfht) for indexing.

In the hash table, the key type is the name's TLV-VALUE, the hash value is the SipHash of the name, and the element type is `FibEntry`.
In case both a real entry and a virtual entry exist at the same name, the virtual entry appears in the hash table, while the real entry is accessible via the `realEntry` pointer.

The `Fib_Lpm` function implements the 2-stage LPM procedure.
It is the only function intended to be called from other packages, and always returns a real entry.
The caller is responsible for obtaining and releasing the RCU read lock.

`FibEntry` carries a sequence number that is incremented upon every insertion.
This allows a PIT entry to save a reference to a FIB entry (`PitEntry_RefreshFibEntry` function) and detect whether the reference is still valid during future retrievals (`PitEntry_FindFibEntry` function).
