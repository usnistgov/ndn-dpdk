# ndn-dpdk/container/fib

This package implements the **Forwarding Information Base (FIB)**.

The FIB is replicated across all NUMA sockets.
The `Fib` type in Go represents the entire FIB, while the `Fib` C struct represents a single replica.

The FIB implements a [2-stage LPM](https://doi.org/10.1109/ANCS.2013.6665203) algorithm for efficient **Longest Prefix Match (LPM)** lookups.

## Go Code

The `Fib` type provides methods to execute read or update commands.
Supported commands include:

* Exact match lookup.
* Reading counters.
* Inserting or replacing an entry.
* Erasing an entry.

The FIB uses the [fibtree](./fibtree) package to organize FIB entries in a name hierarchy.
Read commands are fulfilled in this tree.

Update commands are executed, sequentially, in these steps:

1. Validate command parameters.
2. Apply the update to the tree, which determines what should be inserted and deleted in every replica.
3. Locate old entries in each replica.
4. Allocate new entries in each replica.
   If allocation fails, revert the update in the tree.
5. Insert or replace new entries in each replica.
6. Release the memory of old entries via RCU.

The FIB uses the [fibreplica](./fibreplica) package to access replicas that are implemented in C.

## C Code

The `FibEntry` struct represents either a *real entry* or a *virtual entry*.
A real entry has `height` set to zero, and must have at least one nexthop and a reference to a strategy.
Conversely, a virtual entry has `height` set to a non-zero value and does not have any nexthops.

The `Fib` struct is a thread-safe hash table.
It combines a DPDK mempool for entry allocation with a URCU lock-free resizable RCU hash table (lfht) for indexing.

In the hash table, the key type is the name's TLV-VALUE, the hash value is the SipHash of the name, and the element type is `FibEntry`.
In case both a real entry and a virtual entry exist at the same name, the virtual entry appears in the hash table, while the real entry is accessible via the `realEntry` pointer.

The `Fib_Find` function performs exact match lookups.
The `Fib_Lpm` function implements the 2-stage LPM procedure.
These are the only functions intended to be called from other packages, and always return a real entry.
The caller is responsible for acquiring and releasing the RCU read lock.

`FibEntry` carries a sequence number that is incremented upon every insertion.
This allows a PIT entry to save a reference to a FIB entry (`PitEntry_RefreshFibEntry` function) and detect whether the reference is still valid during future retrievals (`PitEntry_FindFibEntry` function).

The `FibEntryDyn` struct contains counters and strategy scratch area.
Each `FibEntry` contains a vector of `FibEntryDyn`.
Each forwarding thread is assigned one position in this vector, and may update the `FibEntryDyn` without RCU.
