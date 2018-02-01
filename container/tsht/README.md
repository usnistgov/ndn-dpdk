# ndn-dpdk/container/tsht

This directory provides a **Thread-Safe Hash Table (TSHT)**, which is used to implement the [FIB](../fib/).

TSHT combines a DPDK mempool and a URCU lock-free resizable RCU hash table (lfht).
To insert an entry, the caller should allocate a node from the mempool via `Tsht_Alloc` or `Tsht_AllocT`, and then insert it into the lfht with `Tsht_Insert`.
To find an entry with exact-match query, the caller may use `Tsht_Find` or `Tsht_FindT`.
To erase an entry, the caller should invoke `Tsht_Erase`.
The caller must obtain a URCU read-side lock before invoking any of these functions, and release the lock when it no longer references the entry.
These operations are thread-safe.

There is no Go binding for TSHT. FIB C code directly invokes C functions in this directory.
Unit testing is in FIB package.
