package diskstore

/*
#include "../../csrc/diskstore/diskstore.h"
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"

	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/pktmbuf"
	"ndn-dpdk/dpdk/ringbuffer"
	"ndn-dpdk/ndn"
	"ndn-dpdk/spdk"
)

const BLOCK_SIZE = int(C.DISK_STORE_BLOCK_SIZE)

// Disk-backed Data Store.
type DiskStore struct {
	c  *C.DiskStore
	bd *spdk.Bdev
	th *spdk.Thread
}

// Create a DiskStore.
func New(bdi spdk.BdevInfo, th *spdk.Thread, mp *pktmbuf.Pool, nBlocksPerSlot int) (store *DiskStore, e error) {
	if bdi.GetBlockSize() != int(C.DISK_STORE_BLOCK_SIZE) {
		return nil, fmt.Errorf("bdev block size must be %d", C.DISK_STORE_BLOCK_SIZE)
	}

	store = new(DiskStore)
	store.th = th
	if store.bd, e = spdk.OpenBdev(bdi, spdk.BDEV_MODE_READ_WRITE); e != nil {
		return nil, e
	}

	numaSocket := th.GetLCore().GetNumaSocket()
	store.c = (*C.DiskStore)(eal.Zmalloc("DiskStore", C.sizeof_DiskStore, numaSocket))
	store.c.th = (*C.struct_spdk_thread)(th.GetPtr())
	store.c.bdev = (*C.struct_spdk_bdev_desc)(store.bd.GetPtr())
	store.c.mp = (*C.struct_rte_mempool)(mp.GetPtr())
	store.c.nBlocksPerSlot = C.uint64_t(nBlocksPerSlot)
	store.c.blockSize = C.uint32_t(bdi.GetBlockSize())
	th.Call(func() { store.c.ch = C.spdk_bdev_get_io_channel(store.c.bdev) })
	return store, nil
}

func (store *DiskStore) Close() error {
	store.th.Call(func() { C.spdk_put_io_channel(store.c.ch) })
	eal.Free(store.c)
	return store.bd.Close()
}

func (store *DiskStore) GetSlotIdRange() (min, max uint64) {
	return 1, uint64(store.bd.GetInfo().CountBlocks()/int(store.c.nBlocksPerSlot) - 1)
}

// Asynchronously store a Data packet.
func (store *DiskStore) PutData(slotId uint64, data *ndn.Data) {
	C.DiskStore_PutData(store.c, C.uint64_t(slotId), (*C.Packet)(data.GetPacket().GetPtr()))
}

// Retrieve a Data packet and wait for completion.
func (store *DiskStore) GetData(slotId uint64, dataLen int, interest *ndn.Interest) (data *ndn.Data, e error) {
	var reply *ringbuffer.Ring
	if reply, e = ringbuffer.New(fmt.Sprintf("DiskStoreGetData%x", slotId), 64, eal.NumaSocket{},
		ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti); e != nil {
		return nil, e
	}
	defer reply.Close()

	interestPtr := interest.GetPacket().GetPtr()
	C.DiskStore_GetData(store.c, C.uint64_t(slotId), C.uint16_t(dataLen), (*C.Packet)(interestPtr), (*C.struct_rte_ring)(reply.GetPtr()))

	for {
		pkts := make([]*ndn.Packet, 1)
		n := reply.Dequeue(pkts)
		if n != 1 {
			runtime.Gosched()
			continue
		}
		pkt := pkts[0]
		if pkt.GetPtr() != interestPtr {
			panic("unexpected packet in reply ring")
		}

		interest = pkt.AsInterest()
		interestC := (*C.PInterest)(interest.GetPInterestPtr())
		if uint64(interestC.diskSlotId) != slotId {
			panic("unexpected PInterest.diskSlotId")
		}
		if interestC.diskData != nil {
			data = ndn.PacketFromPtr(unsafe.Pointer(interestC.diskData)).AsData()
		}
		return data, nil
	}
}
