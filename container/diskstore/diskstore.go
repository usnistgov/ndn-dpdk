package diskstore

/*
#include "../../csrc/diskstore/diskstore.h"
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/spdk/bdev"
	"github.com/usnistgov/ndn-dpdk/spdk/spdkenv"
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
	bdi := device.GetInfo()
	if bdi.GetBlockSize() != BlockSize {
		return nil, fmt.Errorf("bdev block size must be %d", BlockSize)
	}

	store = new(DiskStore)
	store.th = th
	if store.bd, e = bdev.Open(device, bdev.ReadWrite); e != nil {
		return nil, e
	}

	numaSocket := th.GetLCore().NumaSocket()
	store.c = (*C.DiskStore)(eal.Zmalloc("DiskStore", C.sizeof_DiskStore, numaSocket))
	store.c.th = (*C.struct_spdk_thread)(th.GetPtr())
	store.c.bdev = (*C.struct_spdk_bdev_desc)(store.bd.GetPtr())
	store.c.mp = (*C.struct_rte_mempool)(mp.GetPtr())
	store.c.nBlocksPerSlot = C.uint64_t(nBlocksPerSlot)
	store.c.blockSize = C.uint32_t(bdi.GetBlockSize())
	th.Call(func() { store.c.ch = C.spdk_bdev_get_io_channel(store.c.bdev) })
	return store, nil
}

// Close closes this DiskStore.
func (store *DiskStore) Close() error {
	store.th.Call(func() { C.spdk_put_io_channel(store.c.ch) })
	eal.Free(store.c)
	return store.bd.Close()
}

// GetSlotIdRange returns a range of possible slot numbers.
func (store *DiskStore) GetSlotIdRange() (min, max uint64) {
	return 1, uint64(store.bd.GetInfo().CountBlocks()/int(store.c.nBlocksPerSlot) - 1)
}

// PutData asynchronously stores a Data packet.
func (store *DiskStore) PutData(slotID uint64, data *ndni.Data) {
	C.DiskStore_PutData(store.c, C.uint64_t(slotID), (*C.Packet)(data.GetPacket().GetPtr()))
}

// GetData retrieves a Data packet from specified slot and waits for completion.
func (store *DiskStore) GetData(slotID uint64, dataLen int, interest *ndni.Interest) (data *ndni.Data, e error) {
	var reply *ringbuffer.Ring
	if reply, e = ringbuffer.New(fmt.Sprintf("DiskStoreGetData%x", slotID), 64, eal.NumaSocket{},
		ringbuffer.ProducerMulti, ringbuffer.ConsumerMulti); e != nil {
		return nil, e
	}
	defer reply.Close()

	interestPtr := interest.GetPacket().GetPtr()
	C.DiskStore_GetData(store.c, C.uint64_t(slotID), C.uint16_t(dataLen), (*C.Packet)(interestPtr), (*C.struct_rte_ring)(reply.GetPtr()))

	for {
		pkts := make([]*ndni.Packet, 1)
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
		if uint64(interestC.diskSlotId) != slotID {
			panic("unexpected PInterest.diskSlotId")
		}
		if interestC.diskData != nil {
			data = ndni.PacketFromPtr(unsafe.Pointer(interestC.diskData)).AsData()
		}
		return data, nil
	}
}
