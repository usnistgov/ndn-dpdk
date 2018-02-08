# ndn-dpdk/container/pcct

This package implements the **PIT-CS Composite Table (PCCT)**.

PCCT is a non-thread-safe hash table that carries both the Pending Interest Table (PIT) and the Content Store (CS).
PCCT combines a DPDK mempool for `PccEntry` allocation, a name hash table, and a token hash table.
Each `PccEntry` carries *either* a PIT entry or a CS entry, but cannot carry both at the same time.

C code for [PIT](../pit/) and [CS](../cs/) is in this directory to avoid circular dependency problems, but their Go bindings and documentation are in their own packages.

## Name hash table

The **name hash table** is indexed by Interest/Data and forwarding hint delegation names.
It uses [uthash v2.0.2](https://troydhanson.github.io/uthash/) library, and stores the head in `PcctPriv` type `keyHt` field.
DPDK hash library is unsuitable for this hash table because it requires fixed length keys, but names are variable length, and padding them to the maximum length and comparing them in full would be too time consuming.

The `PccKey` type represents an index key of the name hash table,
For a PIT entry, `PccKey` contains the Interest name and, if the Interest has a forwarding hint, the name of the chosen delegation.
For a CS entry, `PccKey` contains the Data name and, if the Interest used to retrieve this Data has a forwarding hint, the name of the chosen delegation.
The `PccKey` type stores a copy of Interest/Data name and delegation name.

Searching the name hash table uses `PccSearch` type instead of `PccKey` type.
`PccSearch` contains pointers to linearized names.
This avoids copying these names if they are not fragmented in the input packet.
Searching with a different type is made possible by overriding the `uthash_memcmp` hook with `PccKey_MatchSearchKey` function.

## Token hash table
