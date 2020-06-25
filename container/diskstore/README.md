# ndn-dpdk/container/diskstore

This package contains the **Disk-backed Data Store (DiskStore)** that implements an on-disk storage of Data packets.

The DiskStore is backed by an SPDK block device (bdev), whose block size must be 512 bytes.
One or more adjacent blocks are joined together to form a *slot*, identified by a slot number.
The `DiskStore` type stores and retrieves a Data packet via its slot number.
It does not have an index, and cannot search Data packets by name.

## Use Case

The intended use case of the DiskStore is to extend the Content Store with additional capacity.

When the CS evicts an entry from memory, it may allocate a slot number and record it on the CS entry, and pass the Data to `DiskStore_PutData`.
The DiskStore will write the Data to the assigned slot, and release its mbuf.
Write failures are silently ignored.

When a future Interest matches a CS entry that has no associated packet but a slot number, the forwarding core will pass the Interest and the slot number to `DiskStore_GetData`.
The DiskStore will read the Data from the provided slot into a new mbuf, and return it back to forwarding via a ring buffer.
Specifically, the Interest packet is enqueued, and its `PInterest` struct has a non-zero `diskSlotId`, and the Data packet is assigned to the `diskData` field; in case of a read/parse failure, the `diskData` field is set to `NULL`.
The forwarding core should then re-process the Interest, and use the Data only if the Interest matches the same CS entry again.

Multiple CS instances can share the same DiskStore if they use disjoint ranges of slots.
The CS is responsible for allocating and freeing slot numbers.
It is unnecessary for the CS to inform the DiskStore when the Data in a slot is no longer needed: the CS can simply overwrite that slot with another Data packet when the time comes.
