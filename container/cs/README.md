# ndn-dpdk/container/cs

This package implements the **Content Store (CS)**.

CS is part of the [PIT-CS Composite Table (PCCT)](../pcct/).
PCCT provides storage and lookup functions for CS.

CS's APIs are tied to the PIT.
`Pit_Insert` attempts to insert a PIT entry, but if a CS entry is found on the PCC entry and `__Cs_MatchInterest` determines that the CS entry can satisfy the incoming Interest, it returns the CS entry without inserting a PIT entry, effectively performing a CS lookup.
`Cs_Insert` requires a PIT find result, and the new CS entry would take the place of satisfied PIT entries.

## Prefix and Full Name Match via Indirect Entries

In addition to exact match lookups, CS supports a limited form of non-exact match lookups: a cached Data can be found when the new Interest name is same as the previous Interest name that retrieved the Data.
This allows CS matching with a prefix name, under the assumption that a consumer application would use a consistent prefix to perform name discovery.
It also allows matching with a full name including implicit digest.

When an incoming Data satisfies an Interest with a prefix name or a full name, the CS inserts two entries: a **direct entry** with the Data name as its key, contains the Data, and can match future Interests with the same name as the Data; an **indirect entry** with the Interest name as its key, points to the direct entry, and can match future Interests with the same name as the previous Interest.

A direct entry keeps track of dependent indirect entries.
When CS evicts or erases a direct entry, dependent indirect entries are erased automatically.
Each direct entry can track up to four indirect entries; no more indirect entries could be inserted after this limit is reached.

## Eviction

CS has its own capacity limits, in addition to the capacity limit of PCCT's underlying mempool.
Direct entries and indirect entries are organized separately and have separate capacity limits.

### Indirect Entries: Least Recently Used (LRU)

The cache replacement policy for indirect entries is **LRU-1**.
`CsList` type implements this policy by placing all indirect entries on a doubly linked list.

Rear end of this list is the most recently used entry.
When CS inserts an indirect entry, it is appended to the list's rear end.
When an indirect entry is found during a lookup, it is moved to the list's rear end.

Front end of this list is the least recently used entry.
After an insertion causes the list to exceed its capacity limit, some entries are evicted from its front end.
Although evicting just one entry is sufficient to return the list under the capacity limit, the procedure evicts a bulk of entries for better performance.
As a result, the minimum CS capacity is the eviction bulk size.

### Direct Entries: Adaptive Replacement Cache (ARC)

The cache replacement policy for direct entries is **ARC(c,0)**, where *c* is the configured cache capacity.
`CsArc` type implements the [ARC algorithm](https://www.usenix.org/conference/fast-03/arc-self-tuning-low-overhead-replacement-cache).

ARC is originally designed as a page caching algorithm on a storage system.
Its input is a request stream indicating which pages are being requested.
It also contains a synchronous operation to fetch a page into the cache.
However, synchronous fetching is not possible in an NDN forwarder.
Therefore, instead of treating an incoming Interest as a request, this implementation treats both insertion and successful matching as a request.

ARC's four LRU lists are implemented using `CsList` type.
T1 and T2 contain actual cache entries that have Data packets.
B1 and B2 are *ghost* lists that track history of recently evicted cache entries.
Since an entry in B1 or B2 lacks a Data packet, when it's found during a CS lookup, `__Cs_MatchInterest` would report it as non-match.

`CsArc` also has a fifth DEL list that contains entries no longer needed by ARC.
When ARC algorithm deletes an entry, instead of releasing the entry and dependent indirect entries right away, the entry is moved to the DEL list for bulk deletion later; if the entry was in T1 or T2, its Data packet is released immediately.
CS triggers bulk deletion from the DEL list when list size reaches the eviction bulk size.
As a result, the CS may have up to *2c + CS\_EVICT\_BULK* entries, but no more than *c* Data packets.
