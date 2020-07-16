# ndn-dpdk/container/pcct

This package implements the **PIT-CS Composite Table (PCCT)**, a non-thread-safe hash table that carries both the Pending Interest Table (PIT) and the Content Store (CS).

Each **PCC entry** contains PIT and CS entries for a combination of Interest/Data name and chosen delegation name in the forwarding hint.
It can store a PIT entry for *MustBeFresh=0* (denoted "PitEntry0"), a PIT entry for *MustBeFresh=1* (denoted "PitEntry1"), and a CS entry.
The main `PccEntry` type has one `PccSlot` providing room for one PIT entry or one CS entry.
A `PccEntryExt` can be allocated from the PCCT's mempool to provide two additional slots.
Regardless of available slots, each PCC entry can only have one each of PitEntry0, PitEntry1, and CS entry.

The C code for the [PIT](../pit) and the [CS](../cs) is also part of the [csrc/pcct](../../csrc/pcct) directory, in order to avoid circular dependencies, but the Go bindings and related documentation for each module are in their own separate packages.

## Name Hash Table

The **name hash table** is indexed by Interest/Data and forwarding hint delegation names.
It uses the [uthash](https://troydhanson.github.io/uthash/) library and stores the hash table head in the `keyHt` field of `PcctPriv`.
Hash table expansion has been disabled to provide more predictable performance.
DPDK's hash library is unsuitable for this hash table because it requires fixed-length keys, but NDN names are variable length, and padding them to a fixed maximum length would be too wasteful.

The `PccKey` type represents an index key of the name hash table.
For a PIT entry, `PccKey` contains the Interest name and, if the Interest carries a forwarding hint, the name of the chosen delegation.
For a CS entry, `PccKey` contains the Data name and, if the Interest used to retrieve this Data carried a forwarding hint, the name of the chosen delegation.
Interest/Data name and delegation name are copied into the `PccKey`, which has room for short and medium-length names.
If the names are too long to fit into the `PccKey` itself, `PccKeyExt` objects can be allocated from the PCCT's mempool to provide extra room.
Copying the names, rather than referencing them from the packet (usually an Interest), ensures the name buffers are NUMA-local and allows the packet buffer holding the Interest to be freed when the Interest is satisfied.

Searching the name hash table uses the `PccSearch` type instead of the `PccKey` type.
`PccSearch` contains pointers to linearized names.
Searching with a different type is made possible by customizing uthash's `HASH_KEYCMP` hook to use the `PccKey_MatchSearch` function instead of the built-in definition.

Calling code must supply the hash values.
The hash value for an index key without a forwarding hint is `LName_ComputeHash(name)`.
The hash value for an index key with a forwarding hint is `LName_ComputeHash(name) ^ LName_ComputeHash(delegation)`.

## Token Hash Table

Each PCC entry can have an optional 48-bit token.
The **token hash table** is indexed by this token.
It uses DPDK's hash library, where the index key is the token padded to `uint64_t`, the hash value is in the lower 32 bits of the token, and the user data field is a `PccEntry` pointer.

A newly-inserted PCC entry does not have a token.
Calling code must invoke `Pcct_AddToken` to assign a token, and then it can invoke `Pcct_FindByToken` to retrieve that entry by token.
