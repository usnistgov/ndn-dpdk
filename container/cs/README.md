# ndn-dpdk/container/cs

This package implements the **Content Store (CS)**.

The CS is part of the [PIT-CS Composite Table (PCCT)](../pcct).
The PCCT provides the underlying storage and lookup functions for the CS.

The Content Store APIs are tied to the PIT.
`Pit_Insert` attempts to insert a PIT entry, but if a CS entry is found on the PCC entry and `Cs_MatchInterest_` determines that the CS entry can satisfy the incoming Interest, the CS entry will be returned without inserting a PIT entry, effectively performing a CS lookup.
`Cs_Insert` requires a PIT lookup result, and the inserted CS entry will take the place of the satisfied PIT entries.

## Prefix and Full Name Match via Indirect Entries

In addition to exact-match lookup, the CS supports a limited form of prefix-match lookup: cached Data can be found when an incoming Interest's name is equal to the name of the Interest that originally brought the Data into the CS.
This allows CS matching with a prefix of the Data name, under the assumption that a consumer application uses a consistent prefix to perform name discovery.
It also allows matching with a full name including the implicit digest.

When an incoming Data packet satisfies an Interest with a prefix name or a full name, the CS inserts two entries:

* a **direct entry** with the Data name as its key, it contains the actual Data packet and can match future Interests with the same name as the Data;
* an **indirect entry** with the Interest name as its key, it points to the corresponding direct entry and can match future Interests with the same name as the previous Interest.

A direct entry keeps track of the dependent indirect entries.
When a direct entry is evicted or erased, its dependent indirect entries are automatically erased as well.
Each direct entry can track up to four indirect entries; no more indirect entries can be inserted after this limit is reached.

## Eviction

The CS has its own capacity limits, in addition to the capacity limit of the PCCT's underlying mempool.
Direct entries and indirect entries are organized separately and have separate capacity limits.

### Indirect Entries: Least Recently Used (LRU)

The cache replacement policy for indirect entries is **LRU-1**.
The `CsList` type implements this policy by placing all indirect entries on a doubly linked list.

The rear end of this list is the most recently used entry.
When the CS inserts an indirect entry, it is appended to the list's rear end.
When an indirect entry is found during a lookup, it is moved to the list's rear end.

The front end of this list is the least recently used entry.
After an insertion causes the list to exceed its capacity limit, some entries are evicted from its front end.
Although evicting just one entry would be sufficient to return the list under its capacity limit, the implemented algorithm evicts several entries in bulk for better performance.
As a result, the minimum CS capacity is the eviction bulk size.

### Direct Entries: Adaptive Replacement Cache (ARC)

The cache replacement policy for direct entries is **ARC(c, 0)**, where *c* is the configured cache capacity.
The `CsArc` type implements the [ARC algorithm](https://www.usenix.org/conference/fast-03/arc-self-tuning-low-overhead-replacement-cache).

ARC is originally designed as a page caching algorithm on a storage system.
Its input is a request stream indicating which pages are being requested.
It also contains a synchronous operation to fetch a page into the cache.
However, synchronous fetching is not possible in an NDN forwarder.
Therefore, instead of treating an incoming Interest as a request, this implementation treats both insertion and successful match as a request.

ARC's four LRU lists are implemented using the `CsList` type.
T1 and T2 contain the actual cache entries that have Data packets.
B1 and B2 are *ghost* lists that track the history of recently evicted cache entries.
Since an entry in B1 or B2 lacks a Data packet, when it is found during a CS lookup, `Cs_MatchInterest_` will report it as non-match.

`CsArc` also has a fifth DEL list that contains entries no longer needed by ARC.
When the ARC algorithm decides to delete an entry, instead of releasing it and all dependent indirect entries right away, the entry is moved to the DEL list for bulk deletion later; if the entry was in T1 or T2, its Data packet is released immediately.
The CS triggers bulk deletion from the DEL list when the list size reaches the eviction bulk size.
As a result, the CS may hold up to *2c + CS_EVICT_BULK* entries at any given time, but no more than *c* Data packets.
