// Package disk provides a disk-based Data packet store.
package disk

/*
#include "../../csrc/disk/store.h"

extern int go_getDataCallback(Packet* npkt, uintptr_t ctx);
*/
import "C"
import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/bdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/spdkenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
)

var logger = logging.New("disk")

// BlockSize is the supported bdev block size.
const BlockSize = C.DiskStore_BlockSize

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
	c  *C.DiskStore
	bd *bdev.Bdev
	th *spdkenv.Thread

	getDataCbRevoke func()
	getDataGo       bool
}

// Ptr returns *C.DiskStore pointer.
func (store *Store) Ptr() unsafe.Pointer {
	return unsafe.Pointer(store.c)
}

// SlotRange returns a range of possible slot numbers.
func (store *Store) SlotRange() (min, max uint64) {
	return 1, uint64(store.bd.DevInfo().CountBlocks()/int64(store.c.nBlocksPerSlot) - 1)
}

// PutData asynchronously stores a Data packet.
func (store *Store) PutData(slotID uint64, data *ndni.Packet) (sp bdev.StoredPacket, e error) {
	spC := (*C.BdevStoredPacket)(eal.Zmalloc("BdevStoredPacket", C.sizeof_BdevStoredPacket, eal.NumaSocket{}))
	defer eal.Free(spC)
	npkt := (*C.Packet)(data.Ptr())
	if ok := C.DiskStore_PutPrepare(store.c, npkt, spC); !ok {
		return sp, errors.New("prepare failed")
	}
	C.DiskStore_PutData(store.c, C.uint64_t(slotID), npkt, spC)
	sp = *bdev.StoredPacketFromPtr(unsafe.Pointer(spC))
	return sp, nil
}

// GetData retrieves a Data packet from specified slot and waits for completion.
// This can be used only if the Store was created with StoreGetDataGo.
func (store *Store) GetData(slotID uint64, interest *ndni.Packet, dataBuf *pktmbuf.Packet, sp bdev.StoredPacket) (data *ndni.Packet) {
	if !store.getDataGo {
		logger.Panic("Store is not created with StoreGetDataGo, cannot GetData")
	}

	interestC := (*C.Packet)(interest.Ptr())
	pinterest := C.Packet_GetInterestHdr(interestC)

	reply := make(chan struct{})
	_, dup := getDataReplyMap.LoadOrStore(interestC, reply)
	if dup {
		logger.Panic("ongoing GetData on the same mbuf")
	}

	C.DiskStore_GetData(store.c, C.uint64_t(slotID), interestC,
		(*C.struct_rte_mbuf)(dataBuf.Ptr()), (*C.BdevStoredPacket)(sp.Ptr()))
	<-reply

	if retSlot := uint64(pinterest.diskSlot); retSlot != slotID {
		logger.Panic("unexpected PInterest.diskSlot",
			zap.Uint64("request-slot", slotID),
			zap.Uint64("return-slot", slotID),
		)
	}
	return ndni.PacketFromPtr(unsafe.Pointer(pinterest.diskData))
}

func (store *Store) finishPendingTasks() {
	for {
		if cptr.Call(store.th.Post, func() bool {
			if C.rte_hash_count(store.c.requestHt) > 0 {
				return false
			}
			C.spdk_put_io_channel(store.c.ch)
			store.c.ch = nil
			return true
		}).(bool) {
			break
		}
	}
}

// Close closes this Store.
// The SPDK thread must still be active.
func (store *Store) Close() error {
	if store.c.ch != nil {
		store.finishPendingTasks()
	}
	store.getDataCbRevoke()
	eal.Free(store.c.requestArray)
	C.rte_hash_free(store.c.requestHt)
	eal.Free(store.c)
	store.c = nil
	return store.bd.Close()
}

// NewStore creates a Store.
func NewStore(device bdev.Device, th *spdkenv.Thread, nBlocksPerSlot int, getDataCb cptr.Function) (store *Store, e error) {
	bdi := device.DevInfo()
	if blockSz := bdi.BlockSize(); blockSz != BlockSize {
		return nil, fmt.Errorf("bdev not supported: block size is %d, not %d", blockSz, BlockSize)
	}
	if writeUnit := bdi.WriteUnitSize(); writeUnit != 1 {
		return nil, fmt.Errorf("bdev not supported: write unit size is %d, not 1", writeUnit)
	}

	store = &Store{th: th}
	capacity := math.MaxInt64(256, math.MinInt64(bdi.CountBlocks()/1024, 8192))
	socket := th.LCore().NumaSocket()

	if store.bd, e = bdev.Open(device, bdev.ReadWrite); e != nil {
		return nil, e
	}

	store.c = (*C.DiskStore)(eal.Zmalloc("DiskStore", C.sizeof_DiskStore, socket))
	store.bd.CopyToC(unsafe.Pointer(&store.c.bdev))
	store.c.nBlocksPerSlot = C.uint64_t(nBlocksPerSlot)
	store.c.th = (*C.struct_spdk_thread)(th.Ptr())

	htID := C.CString(eal.AllocObjectID("disk.Store.ht"))
	defer C.free(unsafe.Pointer(htID))
	if store.c.requestHt = C.HashTable_New(C.struct_rte_hash_parameters{
		name:       htID,
		entries:    C.uint32_t(capacity),
		key_len:    C.sizeof_uint64_t,
		socket_id:  C.int(socket.ID()),
		extra_flag: C.RTE_HASH_EXTRA_FLAGS_EXT_TABLE,
	}); store.c.requestHt == nil {
		e := eal.GetErrno()
		store.bd.Close()
		eal.Free(store.c)
		return nil, fmt.Errorf("HashTable_New failed: %w", e)
	}
	store.c.requestArray = (*C.DiskStoreRequest)(eal.Zmalloc("disk.Store.requestArray",
		C.sizeof_DiskStoreRequest*(1+C.rte_hash_max_key_id(store.c.requestHt)), socket))

	f, ctx, revoke := StoreGetDataCallback.CallbackReuse(getDataCb)
	store.c.getDataCb, store.c.getDataCtx, store.getDataCbRevoke = C.DiskStore_GetDataCb(f), C.uintptr_t(ctx), revoke
	store.getDataGo = getDataCb == StoreGetDataGo

	logger.Info("DiskStore ready",
		zap.Uintptr("store", uintptr(unsafe.Pointer(store.c))),
		zap.String("bdev", store.bd.DevInfo().Name()),
	)
	return store, nil
}
