package diskstore

/*
#include "../../csrc/diskstore/diskstore.h"
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// BlockSize is the supported bdev block size.
const BlockSize = int(C.DISK_STORE_BLOCK_SIZE)

// DiskStore represents a disk-backed Data Store.
type DiskStore struct {
	c  *C.DiskStore
	bd *bdev.Bdev
	th *spdkenv.Thread
}

// New creates a DiskStore.
func New(device bdev.Device, th *spdkenv.Thread, mp *pktmbuf.Pool, nBlocksPerSlot int) (store *DiskStore, e error) {
	bdi := device.DevInfo()
	if bdi.BlockSize() != BlockSize {
		return nil, fmt.Errorf("bdev block size must be %d", BlockSize)
	}

	store = new(DiskStore)
	store.th = th
	if store.bd, e = bdev.Open(device, bdev.ReadWrite); e != nil {
		return nil, e
	}

	socket := th.LCore().NumaSocket()
	store.c = (*C.DiskStore)(eal.Zmalloc("DiskStore", C.sizeof_DiskStore, socket))
	store.c.th = (*C.struct_spdk_thread)(th.Ptr())
	store.c.bdev = (*C.struct_spdk_bdev_desc)(store.bd.Ptr())
	store.c.mp = (*C.struct_rte_mempool)(mp.Ptr())
	store.c.nBlocksPerSlot = C.uint64_t(nBlocksPerSlot)
	store.c.blockSize = C.uint32_t(bdi.BlockSize())
	cptr.Call(th.Post, func() { store.c.ch = C.spdk_bdev_get_io_channel(store.c.bdev) })
	return store, nil
}

// Close closes this DiskStore.
func (store *DiskStore) Close() error {
	cptr.Call(store.th.Post, func() { C.spdk_put_io_channel(store.c.ch) })
	eal.Free(store.c)
	return store.bd.Close()
}

// SlotRange returns a range of possible slot numbers.
func (store *DiskStore) SlotRange() (min, max uint64) {
	return 1, uint64(store.bd.DevInfo().CountBlocks()/int(store.c.nBlocksPerSlot) - 1)
}

// PutData asynchronously stores a Data packet.
func (store *DiskStore) PutData(slotID uint64, data *ndni.Packet) {
	C.DiskStore_PutData(store.c, C.uint64_t(slotID), (*C.Packet)(data.Ptr()))
}

// GetData retrieves a Data packet from specified slot and waits for completion.
func (store *DiskStore) GetData(slotID uint64, dataLen int, interest *ndni.Packet) (data *ndni.Packet, e error) {
	var reply *ringbuffer.Ring
	if reply, e = ringbuffer.New(64, eal.NumaSocket{},
		ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti); e != nil {
		return nil, e
	}
	defer reply.Close()

	interestC := (*C.Packet)(interest.Ptr())
	pinterest := C.Packet_GetInterestHdr(interestC)
	C.DiskStore_GetData(store.c, C.uint64_t(slotID), C.uint16_t(dataLen), interestC, (*C.struct_rte_ring)(reply.Ptr()))

	pkts := make([]*ndni.Packet, 1)
	for {
		if reply.Dequeue(pkts) == 1 {
			break
		}
		runtime.Gosched()
	}
	pkt := pkts[0]
	if pkt != interest {
		panic("unexpected packet in reply ring")
	}

	if uint64(pinterest.diskSlot) != slotID {
		panic("unexpected PInterest.diskSlotId")
	}
	data = ndni.PacketFromPtr(unsafe.Pointer(pinterest.diskData))
	return data, nil
}
