# ndn-dpdk/container/pcct

This package implements the **PIT-CS Composite Table (PCCT)**, a non-thread-safe hash table that carries both the Pending Interest Table (PIT) and the Content Store (CS).

Each **PCC entry** contains PIT and CS entries for a combination of Interest/Data name and chosen delegation name in the forwarding hint.
It can store a PIT entry for MustBeFresh=0 (denoted "PitEntry0"), a PIT entry for MustBeFresh=1 (denoted "PitEntry1"), and a CS entry.
The main `PccEntry` type has one `PccSlot` providing room for one PIT entry or one CS entry.
An `PccEntryExt` can be allocated from the PCCT's mempool to provide two additional slots.
Regardless of available slots, each PCC entry can only have one each of PitEntry0, PitEntry1, and CS entry.

C code for [PIT](../pit/) and [CS](../cs/) is in this directory to avoid circular dependency problems, but their Go bindings and documentation are in their own packages.

## Name hash table

The **name hash table** is indexed by Interest/Data and forwarding hint delegation names.
It uses [uthash v2.0.2](https://troydhanson.github.io/uthash/) library, and stores the head in `PcctPriv` type `keyHt` field.
DPDK hash library is unsuitable for this hash table because it requires fixed-length keys, but names are variable length, and padding them to the maximum length would be too time consuming.

The `PccKey` type represents an index key of the name hash table,
For a PIT entry, `PccKey` contains the Interest name and, if the Interest has a forwarding hint, the name of the chosen delegation.
For a CS entry, `PccKey` contains the Data name and, if the Interest used to retrieve this Data has a forwarding hint, the name of the chosen delegation.
Interest/Data name and delegation name are copied into `PccKey`, which has room for short and medium length names.
If the names are too long to fit into `PccKey` itself, `PccKeyExt` objects can be allocated from the PCCT's mempool to provide extra room.
Copying the names, rather than referencing them from the packet (usually an Interest), ensures the name buffers are NUMA-local, and allows the packet buffer holding the Interest to be freed when the Interest is satisfied.

Searching the name hash table uses `PccSearch` type instead of `PccKey` type.
`PccSearch` contains pointers to linearized names.
This avoids copying these names if they are not fragmented in the input packet.
Searching with a different type is made possible by overriding the `uthash_memcmp` hook with `PccKey_MatchSearchKey` function.

Calling code must provide hash values.
Hash value for an index key without forwarding hint is `Name_ComputeHash(name)`.
Hash value for an index key with forwarding hint is `Name_ComputeHash(name) ^ Name_ComputeHash(delegation)`.

## Token hash table

Each PCC entry can have an optional 48-bit token.
The **token hash table** is indexed by this token.
It uses DPDK hash library, where the index key is the token padded to `uint64_t`, the hash value is the lower 32 bits of the token, and the user data field is a `PccEntry` pointer.

A newly-inserted PCC entry does not have a token.
Calling code must invoke `Pcct_AddToken` to assign a token, and then it can invoke `Pcct_FindByToken` to retrieve the entry by token.
