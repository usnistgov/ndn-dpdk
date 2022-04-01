# ndn-dpdk/dpdk/bdev

This package contains bindings of SPDK block device (bdev) layer.
It is used by [Disk-based Data Store](../../container/disk), which is intended for extending the forwarder's Content Store with additional capacity.

## Block Device Drivers

SPDK offers a range of block device drivers.
This package supports a subset of these drivers.

**Malloc** type represents a memory-backed emulated block device.
The storage space is reserved from hugepage memory, not on an actual disk drive.
It is mainly for unit testing.

**File** type represents a file-backed virtual block device.
It may use either io\_uring or Linux AIO driver as backend.
The storage space is a file located in the local filesystem.
For best performance, you should keep the file on a local disk, not a network storage.

**Nvme** type represents a hardware NVMe drive.
It contains one or more **NvmeNamespace**s, each is a block device.
Although SPDK NVMe driver supports both local PCIe drives and NVMe over Fabrics, this implementation is limited to local drives only.
SPDK would have exclusive control over the PCI device, which means you cannot use the NVMe driver on the same NVMe device where you have kernel filesystems.
In most cases, the PCI device should be bound to `vfio-pci` kernel driver.
Importantly, once you assign an NVMe device to be used with NDN-DPDK via the SPDK NVMe driver, all existing data will be erased.

**Delay** type represents a virtual block device that wraps an existing block device, adding a delay to each I/O operation.
**ErrorInjection** type represents a virtual block device that wraps an existing block device, injecting an error to certain I/O operations.
They are mainly for unit testing.

All types described above implement the **Device** interface, which allows retrieving information about the block device.

## Open Device and I/O Operations

**Bdev** type represents an `Open()`-ed block device.
This package only supports block devices with 512-octet block size, which is used by the majority of known devices.

I/O operations may be submitted to an open block device.
This package implements two I/O operations:

* `Bdev_WritePacket` writes a packet mbuf to a block offset.
* `Bdev_ReadPacket` reads the packet at a block offset to an mbuf.

All I/O operations are asynchronous.
If multiple I/O operations are accessing the same block offset, it's caller's responsible to ensure proper sequencing.

## Memory Alignment

Certain SPDK drivers have specific memory alignment requirements.
`Bdev_WritePacket` and `Bdev_ReadPacket` can handle these requirements.

Some NVMe devices require every buffers to have dword (32-bit) alignment, and every buffer length to be a multiple of 4.
To satisfy this requirement without copying, the content being written to the block device may contain part of the headroom and tailroom in the mbuf.
The amount of excess writing is captured in a **BdevStoredPacket** struct, which must be saved by the caller.
During reading, the original packet is recovered from the saved information, by either adjusting headroom or `memmove`-ing.
This design chooses to incur less overhead during writing and more overhead during reading, because the primary use case is packet caching, in which only a subset of written packets would generate cache hits and need to be be read back, and each packet reenters in-memory cache after being read.

Both file-backed drivers expect the buffer to be aligned at 512-octet boundary, otherwise the driver will incur copying overhead.
During writing, such copying is mostly unavoidable, because the excess writing method described above would greatly amplify storage usage.
During reading, the mbuf must have sufficient dataroom to cover the packet itself plus twice the block size, so that the read function can find a properly aligned memory address within the dataroom to read the packet.
