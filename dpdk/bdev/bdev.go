// Package bdev contains bindings of SPDK block device layer.
package bdev

/*
#include "../../csrc/dpdk/bdev.h"
#include <spdk/thread.h>

typedef struct go_BdevRequest
{
	uintptr_t handle;
	BdevRequest breq;
	BdevStoredPacket sp;
} go_BdevRequest;

extern void go_bdevEvent(enum spdk_bdev_event_type type, struct spdk_bdev* bdev, uintptr_t ctx);
extern void go_bdevComplete(BdevRequest* breq, int res);

static void c_bdev_io_complete(struct spdk_bdev_io* io, bool success, void* breq)
{
	go_bdevComplete((BdevRequest*)breq, success ? 0 : EIO);
}

static int c_spdk_bdev_unmap_blocks(struct spdk_bdev_desc* desc, struct spdk_io_channel* ch, uint64_t offset_blocks, uint64_t num_blocks, BdevRequest* breq)
{
	return spdk_bdev_unmap_blocks(desc, ch, offset_blocks, num_blocks, c_bdev_io_complete, breq);
}
*/
import "C"
import (
	"fmt"
	"runtime/cgo"
	"strings"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/zyedidia/generic"
	"go.uber.org/zap"
)

var logger = logging.New("bdev")

// Mode indicates mode of opening a block device.
type Mode bool

// Mode values.
const (
	ReadOnly  Mode = false
	ReadWrite Mode = true
)

// StoredPacket describes length and alignment of a stored packet.
type StoredPacket C.BdevStoredPacket

// Ptr returns *C.BdevStoredPacket pointer.
func (sp *StoredPacket) Ptr() unsafe.Pointer {
	return unsafe.Pointer(sp)
}

// StoredPacketFromPtr converts *C.BdevStoredPacket pointer to StoredPacket.
func StoredPacketFromPtr(ptr unsafe.Pointer) *StoredPacket {
	return (*StoredPacket)(ptr)
}

// Bdev represents an open block device descriptor.
type Bdev struct {
	Device
	c  C.Bdev
	ch *C.struct_spdk_io_channel
}

// Close closes the block device.
func (bd *Bdev) Close() error {
	eal.CallMain(func() {
		if bd.ch != nil {
			C.spdk_put_io_channel(bd.ch)
			bd.ch = nil
		}
		C.spdk_bdev_close(bd.c.desc)
	})
	logger.Info("device closed", zap.String("name", bd.DevInfo().Name()))
	return nil
}

// CopyToC copies to *C.Bdev.
func (bd *Bdev) CopyToC(ptr unsafe.Pointer) {
	*(*C.Bdev)(ptr) = bd.c
}

func (bd *Bdev) enableDwordAlign() {
	bd.c.dwordAlign = true
	bd.c.bufAlign = generic.Max(bd.c.bufAlign, 4)
}

func (bd *Bdev) do(pkt *pktmbuf.Packet, f func(breq *C.BdevRequest)) error {
	done := make(chan C.int)
	ctx := cgo.NewHandle(done)
	defer ctx.Delete()

	req := eal.Zmalloc[C.go_BdevRequest]("BdevRequest", C.sizeof_go_BdevRequest, eal.NumaSocket{})
	defer eal.Free(req)
	req.handle = C.uintptr_t(ctx)

	eal.PostMain(cptr.Func0.Void(func() {
		if bd.ch == nil {
			bd.ch = C.spdk_bdev_get_io_channel(bd.c.desc)
		}
		req.breq.pkt = (*C.struct_rte_mbuf)(pkt.Ptr())
		req.breq.sp = &req.sp
		req.breq.cb = C.BdevRequestCb(C.go_bdevComplete)
		f(&req.breq)
	}))
	return eal.MakeErrno(<-done)
}

// UnmapBlocks notifies the device that the data in the blocks are no longer needed.
func (bd *Bdev) UnmapBlocks(blockOffset, blockCount int64) error {
	return bd.do(nil, func(breq *C.BdevRequest) {
		res := C.c_spdk_bdev_unmap_blocks(bd.c.desc, bd.ch, C.uint64_t(blockOffset), C.uint64_t(blockCount), breq)
		if res != 0 {
			go_bdevComplete(breq, res)
		}
	})
}

// ReadPacket reads blocks via scatter gather list.
func (bd *Bdev) ReadPacket(blockOffset int64, pkt *pktmbuf.Packet, sp StoredPacket) error {
	return bd.do(pkt, func(breq *C.BdevRequest) {
		*(*StoredPacket)(breq.sp) = sp
		C.Bdev_ReadPacket(&bd.c, bd.ch, C.uint64_t(blockOffset), breq)
	})
}

// WritePacket writes blocks via scatter gather list.
func (bd *Bdev) WritePacket(blockOffset int64, pkt *pktmbuf.Packet) (sp StoredPacket, e error) {
	e = bd.do(pkt, func(breq *C.BdevRequest) {
		if ok := C.Bdev_WritePrepare(&bd.c, breq.pkt, breq.sp); !ok {
			go_bdevComplete(breq, C.EMSGSIZE)
			return
		}
		C.Bdev_WritePacket(&bd.c, bd.ch, C.uint64_t(blockOffset), breq)
		sp = *(*StoredPacket)(breq.sp)
	})
	return
}

// Open opens a block device.
func Open(device Device, mode Mode) (bd *Bdev, e error) {
	bdi := device.DevInfo()
	if blockSize := bdi.BlockSize(); blockSize != RequiredBlockSize {
		return nil, fmt.Errorf("not supported: block size is %d, not %d", blockSize, RequiredBlockSize)
	}
	if writeUnit := bdi.WriteUnitSize(); writeUnit != 1 {
		return nil, fmt.Errorf("not supported: write unit size is %d, not 1", writeUnit)
	}

	bd = &Bdev{Device: device}
	eal.CallMain(func() {
		if res := C.spdk_bdev_open_ext(C.spdk_bdev_get_name(bdi.ptr()), C.bool(mode),
			C.spdk_bdev_event_cb_t(C.go_bdevEvent), nil, &bd.c.desc); res != 0 {
			e = eal.MakeErrno(res)
			return
		}
	})
	if e != nil {
		return nil, e
	}

	bd.c.bufAlign = C.uint32_t(bdi.BufAlign())
	if strings.HasPrefix(bdi.Name(), "nvme") {
		bd.enableDwordAlign()
	}
	logger.Info("device opened",
		zap.Uintptr("desc", uintptr(unsafe.Pointer(bd.c.desc))),
		zap.Bool("dword-align", bool(bd.c.dwordAlign)),
		zap.Inline(bdi),
	)
	return bd, nil
}

// ForceDwordAlign enables DwordAlign mode even if the driver does not require it.
// This is mainly useful for unit testing.
func ForceDwordAlign(bd *Bdev) {
	bd.enableDwordAlign()
}

//export go_bdevEvent
func go_bdevEvent(typ C.enum_spdk_bdev_event_type, bdev *C.struct_spdk_bdev, ctx C.uintptr_t) {
	logger.Info("event",
		zap.Int("event-type", int(typ)),
		zap.String("name", (*Info)(bdev).Name()),
	)
}

//export go_bdevComplete
func go_bdevComplete(breq *C.BdevRequest, res C.int) {
	req := (*C.go_BdevRequest)(unsafe.Add(unsafe.Pointer(breq), -int(unsafe.Offsetof(C.go_BdevRequest{}.breq))))
	done := cgo.Handle(req.handle).Value().(chan C.int)
	done <- res
}
