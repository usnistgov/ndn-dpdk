# ndn-dpdk/container/disk

This package implements an on-disk storage of Data packets.

## Disk-backed Data Store (DiskStore)

DiskStore is an on-disk storage of Data packets.

It is backed by an [SPDK block device (bdev)](../../dpdk/bdev), whose block size must be 512 bytes.
One or more adjacent blocks are joined together to form a *slot*, identified by a slot number.
The `DiskStore` type stores and retrieves a Data packet via its slot number.
It does not have an index, and cannot search Data packets by name.

The intended use case of the DiskStore is to extend the Content Store with additional capacity.

When the CS evicts an entry from memory, it may allocate a slot number and record it on the CS entry, and pass the Data to `DiskStore_PutData`.
The DiskStore will asynchronously write the Data to the assigned slot, and then release its mbuf.
Write failures are silently ignored.

When a future Interest matches a CS entry that has no associated packet but a slot number, the forwarding thread will pass the Interest and the slot number to `DiskStore_GetData`.
The DiskStore will asynchronously read the Data from the provided slot into a new mbuf, and then return it back to forwarder data plane via a callback.
Specifically, the Interest packet is enqueued, its `PInterest` struct has a non-zero `diskSlot`, and the Data packet is assigned to the `diskData` field; in case of a read/parse failure, the `diskData` field is set to `NULL`.
The forwarder data plane should then re-process the Interest, and use the Data only if the Interest matches the same CS entry again.

Multiple CS instances can share the same DiskStore if they use disjoint ranges of slots.
The CS is responsible for allocating and freeing slot numbers.
It is unnecessary for the CS to inform the DiskStore when the Data in a slot is no longer needed: the CS can simply overwrite that slot with another Data packet when the time comes.

Since both PutData and GetData are asynchronous, it's possible for one or more GetData requests to arrive before the PutData on the same slot completes.
DiskStore has a per-slot queue, indexed in a hashtable, to solve this issue.
This ensures all requests on the same slot are processed in the order they are received, so that the forwarding thread does not need to concern the asynchronous nature of disk operations.

## Disk Slot Allocator (DiskAlloc)

DiskAlloc allocates disk slots within a consecutive range of a DiskStore.
Available slots of a DiskStore are statically partitioned and associated with each Content Store instance.
The CS can then using a `DiskAlloc` instance to allocate a disk slot for each Data packet it wants to write to disk.

DiskAlloc is implemented as a bitmap.
If a bit is set to 1, the disk slot is available.
If a bit is cleared to 0, the disk slot is occupied.
