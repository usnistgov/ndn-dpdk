# ndn-dpdk/container/fib

This package implements the **Forwarding Information Base (FIB)**.

In C code, `Fib` data structure is a hash table. It is a customization of [Thread-Safe Hash Table (TSHT)](../tsht/), in which name prefix hashes serve as hash values, and linearized names serve as keys.
Only `Fib_Lpm` is intended to be called from other packages, and the caller is responsible
for obtaining and releasing the RCU read lock.

In Go code, `Fib` type has a pointer to `C.Fib` data structure.
It exports `Insert` and `Erase` methods for FIB updates.
It also exports an `Lpm` method to (inefficiently) perform longest prefix match lookups, which internally obtains and releases the RCU read lock, and copies the FIB entry before returning it.
