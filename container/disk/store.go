// Package disk provides a disk-based Data packet store.
package disk

/*
#include "../../csrc/disk/store.h"

extern int go_getDataCallback(Packet* npkt, uintptr_t ctx);
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// BlockSize is the supported bdev block size.
const BlockSize = C.DISK_STORE_BLOCK_SIZE

var (
	// StoreGetDataCallback is a C function type for store.GetData callback.
	StoreGetDataCallback = cptr.FunctionType{"Packet"}

	// StoreGetDataGo is a StoreGetDataCallback implementation for receiving the Data in Go code.
	StoreGetDataGo = StoreGetDataCallback.C(unsafe.Pointer(C.go_getDataCallback), uintptr(0))

	getDataReplyMap sync.Map
)

//export go_getDataCallback
func go_getDataCallback(npkt *C.Packet, ctx C.uintptr_t) C.int {
	reply, ok := getDataReplyMap.LoadAndDelete(npkt)
	if !ok {
		panic("unexpected invocation")
	}
	close(reply.(chan struct{}))
	return 0
}

// Store represents a disk-backed Data Store.
type Store struct {
	c               *C.DiskStore
	bd              *bdev.Bdev
	th              *spdkenv.Thread
	getDataCbRevoke func()
}

// Ptr returns *C.DiskAlloc pointer.
func (store *Store) Ptr() unsafe.Pointer {
	return unsafe.Pointer(store.c)
}

// Close closes this DiskStore.
func (store *Store) Close() error {
	cptr.Call(store.th.Post, func() {
		if store.c.ch != nil {
			C.spdk_put_io_channel(store.c.ch)
		}
	})
	eal.Free(store.c)
	store.getDataCbRevoke()
	return store.bd.Close()
}

// SlotRange returns a range of possible slot numbers.
func (store *Store) SlotRange() (min, max uint64) {
	return 1, uint64(store.bd.DevInfo().CountBlocks()/int(store.c.nBlocksPerSlot) - 1)
}

// PutData asynchronously stores a Data packet.
func (store *Store) PutData(slotID uint64, data *ndni.Packet) {
	C.DiskStore_PutData(store.c, C.uint64_t(slotID), (*C.Packet)(data.Ptr()))
}

// GetData retrieves a Data packet from specified slot and waits for completion.
// This can be used only if the Store was created with StoreGetDataGo.
func (store *Store) GetData(slotID uint64, interest *ndni.Packet, dataBuf *pktmbuf.Packet) (data *ndni.Packet) {
	interestC := (*C.Packet)(interest.Ptr())
	pinterest := C.Packet_GetInterestHdr(interestC)

	reply := make(chan struct{})
	_, dup := getDataReplyMap.LoadOrStore(interestC, reply)
	if dup {
		panic("ongoing GetData on the same mbuf")
	}

	C.DiskStore_GetData(store.c, C.uint64_t(slotID), interestC, (*C.struct_rte_mbuf)(dataBuf.Ptr()))
	<-reply

	if uint64(pinterest.diskSlot) != slotID {
		panic("unexpected PInterest.diskSlot")
	}
	return ndni.PacketFromPtr(unsafe.Pointer(pinterest.diskData))
}

// NewStore creates a Store.
func NewStore(device bdev.Device, th *spdkenv.Thread, nBlocksPerSlot int, getDataCb cptr.Function) (store *Store, e error) {
	bdi := device.DevInfo()
	if bdi.BlockSize() != BlockSize {
		return nil, fmt.Errorf("bdev block size must be %d", BlockSize)
	}

	store = &Store{
		th: th,
	}
	if store.bd, e = bdev.Open(device, bdev.ReadWrite); e != nil {
		return nil, e
	}

	socket := th.LCore().NumaSocket()
	store.c = (*C.DiskStore)(eal.Zmalloc("DiskStore", C.sizeof_DiskStore, socket))
	store.c.th = (*C.struct_spdk_thread)(th.Ptr())
	store.c.bdev = (*C.struct_spdk_bdev_desc)(store.bd.Ptr())
	store.c.nBlocksPerSlot = C.uint64_t(nBlocksPerSlot)
	store.c.blockSize = C.uint32_t(bdi.BlockSize())

	f, ctx, revoke := StoreGetDataCallback.CallbackReuse(getDataCb)
	store.c.getDataCb, store.c.getDataCtx, store.getDataCbRevoke = C.DiskStore_GetDataCb(f), C.uintptr_t(ctx), revoke

	return store, nil
}
