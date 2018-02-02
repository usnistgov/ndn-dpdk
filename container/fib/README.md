# ndn-dpdk/container/fib

This package implements the **Forwarding Information Base (FIB)**.
This FIB implements [2-stage LPM](http://ieeexplore.ieee.org/document/6665203/) algorithm for efficient Longest Prefix Match (LPM) lookups.

## C code

`Fib` data structure is a hash table. It is a customization of [Thread-Safe Hash Table (TSHT)](../tsht/), in which name prefix hashes serve as hash values, and linearized names serve as keys.
`Fib_Lpm` function implements 2-stage LPM lookup. It is the only function intended to be called from other packages. The caller is responsible for obtaining and releasing the RCU read lock.

## Go code

`Fib` type has a pointer to `C.Fib` data structure.
It exports `Insert` and `Erase` methods for FIB updates.
It also exports `Find` and `Lpm` methods to (inefficiently) perform exact match and longest prefix match lookups.

`Fib` type also contains a tree of FIB entry names, kept in sync with `C.Fib`.
The main purpose of this tree is to compute *MD* used in 2-stage LPM algorithm.

A dedicated goroutine sequentially executes all commands in a registered RCU read-side thread. `Fib` internally obtains and releases RCU read lock, and lookup methods copy the result FIB entry before returning.
