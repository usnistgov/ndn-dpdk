# ndn-dpdk/container/diskstore

This package contains the **Disk-backed Data Store (DiskStore)** that implements an on-disk storage of Data packets.

The DiskStore is backed by an SPDK block device (bdev), whose block size must be 512 bytes.
One or more adjacent blocks are joined together to form a *slot*, identified by a slot number.
`DiskStore` type stores and retrieves a Data packet via its slot number.
It does not have an index, and cannot search Data packets by name.

## Use Case

The intended use case of DiskStore is to extend the Content Store with extra capacity.

When CS evicts an entry from memory, it may allocate a slot number and record it on the CS entry, and pass the Data to `DiskStore_PutData`.
DiskStore will write the Data to the assigned slot, and release its mbuf.
Write failures are silently ignored.

When a subsequent Interest matches an CS entry that has no packet but a slot number, forwarding should pass the Interest and the slot number to `DiskStore_GetData`.
DiskStore will read the Data from the assigned slot into a new mbuf, and return it back to forwarding via a ring buffer.
Specifically, the Interest packet is enqueued, and its `PInterest` struct has non-zero `diskSlotId`, and the Data packet is assigned to `diskData` field; in case of a read/parse failure, `diskData` field is set to NULL.
Forwarding should then re-process the Interest, and use the Data only if the Interest matches the same CS entry again.

Multiple CS instances can share the same DiskStore if they use distinct ranges of slot numbers.
CS is responsible for allocating and freeing slot numbers.
It is unnecessary for CS to inform DiskStore when the Data in a slot is no longer needed: CS can simply overwrite the slot with another Data.
