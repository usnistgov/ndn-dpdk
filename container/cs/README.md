# ndn-dpdk/container/cs

This package implements the **Content Store (CS)**.

CS is part of the [PIT-CS Composite Table (PCCT)](../pcct/).
PCCT provides storage and lookup functions for CS.

CS's APIs are tied to the PIT.
`Pit_Insert` performs CS lookup.
`Cs_Insert` requires a PIT find result, and the new CS entry would take the place of satisfied PIT entries.

## Eviction

CS has its own capacity limit, in addition to the capacity limit of PCCT's underlying mempool.
After an insertion causes the CS to exceed its capacity limit, some entries are evicted.
Although evicting just one entry is sufficient to return the CS under the capacity limit, the procedure evicts a bulk of entries for better performance.
As a result, the minimum CS capacity is the eviction bulk size.

The cache replacement policy is **First-In-First-Out (FIFO)**.
To implement this policy, the CS places all entries on a doubly linked list.
CS insertion procedure appends a new entry or moves a refreshed entry to the tail of this linked list.
CS eviction procedure erases entries from the head of this linked list.

## Prefix Match Support

In addition to exact match lookups, CS supports a limited form of prefix match lookups: a cached Data can be found when the new Interest name is same as the previous Interest name that retrieved the Data.
This design works under the assumption that a consumer application would use a consistent name to perform name discovery.

When an Interest with a prefix name retrieves a Data, the CS inserts two entries: a *direct* entry that has the Data name and encloses the Data, and an *indirect* entry that has the Interest name and points to the direct entry.
The direct entry can match an Interest with the same name as the Data.
The indirect entry can match an Interest with the same name as the previous Interest.

A direct entry keeps track of dependent indirect entries.
Every time an indirect entry is inserted, the direct entry is moved after the new indirect entry in the eviction linked list, preventing the direct entry from being evicted earlier than the indirect entry.
In case a direct entry is erased due to an administrative command, dependent indirect entries are erased automatically.
