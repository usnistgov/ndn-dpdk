# ndn-dpdk/container/cs

This package implements the **Content Store (CS)**.

CS is part of the [PIT-CS Composite Table (PCCT)](../pcct/).
PCCT provides storage and lookup functions for CS.

CS's APIs are tied to the PIT.
`Pit_Insert` performs CS lookup.
`Cs_Insert` requires a PIT find result, and the new CS entry would take the place of satisfied PIT entries.

This CS only supports exact match lookups.

## Eviction

CS has its own capacity limit, in addition to the capacity limit of PCCT's underlying mempool.
After an insertion causes the CS to exceed its capacity limit, some entries are evicted.
Instead evicting just one entry that is sufficient to return the CS under the capacity limit, the procedure evicts a bulk of entries for better performance.
As a result, the minimum CS capacity is the eviction bulk size.

The cache replacement policy is **First-In-First-Out (FIFO)**.
To implement this policy, the CS places all entries on a doubly linked list.
CS insertion procedure appends a new entry or moves a refreshed entry to the tail of this linked list.
CS eviction procedure erases entries from the head of this linked list.
