package bdev

/*
#include "../../csrc/dpdk/bdev.h"
#include <spdk/thread.h>

extern void go_bdevEvent(enum spdk_bdev_event_type type, struct spdk_bdev* bdev, void* ctx);
extern void go_bdevIoComplete(struct spdk_bdev_io* io, bool success, void* ctx);
*/
import "C"
import (
	"errors"
	"io"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
)

// Mode indicates mode of opening a block device.
type Mode bool

// Modes of opening a block device.
const (
	ReadOnly  Mode = false
	ReadWrite Mode = true
)

// Bdev represents an open block device descriptor.
type Bdev struct {
	c         *C.struct_spdk_bdev_desc
	ch        *C.struct_spdk_io_channel
	blockSize int64
	nBlocks   int64
}

// Open opens a block device.
func Open(device Device, mode Mode) (bd *Bdev, e error) {
	bdi := device.DevInfo()
	bd = new(Bdev)
	eal.CallMain(func() {
		if res := C.spdk_bdev_open_ext(C.spdk_bdev_get_name(bdi.ptr()), C.bool(mode),
			C.spdk_bdev_event_cb_t(C.go_bdevEvent), nil, &bd.c); res != 0 {
			e = eal.Errno(res)
			return
		}
		bd.ch = C.spdk_bdev_get_io_channel(bd.c)
	})
	if e != nil {
		return nil, e
	}
	bd.blockSize = int64(bdi.BlockSize())
	bd.nBlocks = int64(bdi.CountBlocks())
	return bd, nil
}

//export go_bdevEvent
func go_bdevEvent(typ C.enum_spdk_bdev_event_type, bdev *C.struct_spdk_bdev, ctx unsafe.Pointer) {
}

// Close closes the block device.
func (bd *Bdev) Close() error {
	eal.CallMain(func() {
		C.spdk_put_io_channel(bd.ch)
		C.spdk_bdev_close(bd.c)
	})
	return nil
}

// Ptr returns *C.struct_bdev_bdev_desc pointer.
func (bd *Bdev) Ptr() unsafe.Pointer {
	return unsafe.Pointer(bd.c)
}

// DevInfo returns Info about this device.
func (bd *Bdev) DevInfo() (bdi *Info) {
	return (*Info)(C.spdk_bdev_desc_get_bdev(bd.c))
}

// ReadBlocks reads blocks of data.
func (bd *Bdev) ReadBlocks(blockOffset, blockCount int64, buf []byte) error {
	if blockOffset < 0 || blockOffset+blockCount >= bd.nBlocks {
		return io.ErrUnexpectedEOF
	}
	sizeofBuf := blockCount * bd.blockSize
	if sizeofBuf > int64(len(buf)) {
		return io.ErrShortBuffer
	}

	bufC := eal.Zmalloc("SpdkBdevBuf", sizeofBuf, eal.NumaSocket{})
	defer eal.Free(bufC)

	done := make(chan error)
	eal.PostMain(cptr.VoidFunction(func() {
		ctx := cptr.CtxPut(done)
		res := C.spdk_bdev_read_blocks(bd.c, bd.ch, bufC, C.uint64_t(blockOffset), C.uint64_t(blockCount),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- eal.Errno(-res)
			cptr.CtxClear(ctx)
		}
	}))
	if e := <-done; e != nil {
		return e
	}

	C.rte_memcpy(unsafe.Pointer(&buf[0]), bufC, C.size_t(sizeofBuf))
	return nil
}

// WriteBlocks writes blocks of data.
func (bd *Bdev) WriteBlocks(blockOffset, blockCount int64, buf []byte) error {
	if blockOffset < 0 || blockOffset+blockCount >= bd.nBlocks {
		return io.ErrShortWrite
	}
	sizeofBuf := blockCount * bd.blockSize
	if sizeofBuf > int64(len(buf)) {
		return io.ErrShortBuffer
	}

	bufC := eal.Zmalloc("SpdkBdevBuf", sizeofBuf, eal.NumaSocket{})
	defer eal.Free(bufC)
	C.rte_memcpy(bufC, unsafe.Pointer(&buf[0]), C.size_t(sizeofBuf))

	done := make(chan error)
	eal.PostMain(cptr.VoidFunction(func() {
		ctx := cptr.CtxPut(done)
		res := C.spdk_bdev_write_blocks(bd.c, bd.ch, bufC, C.uint64_t(blockOffset), C.uint64_t(blockCount),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- eal.Errno(-res)
			cptr.CtxClear(ctx)
		}
	}))
	return <-done
}

// ReadPacket reads blocks via scatter gather list.
func (bd *Bdev) ReadPacket(blockOffset, blockCount int64, pkt pktmbuf.Packet) error {
	done := make(chan error)
	eal.PostMain(cptr.VoidFunction(func() {
		ctx := cptr.CtxPut(done)
		res := C.SpdkBdev_ReadPacket(bd.c, bd.ch, (*C.struct_rte_mbuf)(pkt.Ptr()),
			C.uint64_t(blockOffset), C.uint64_t(blockCount), C.uint32_t(bd.blockSize),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- eal.Errno(-res)
			cptr.CtxClear(ctx)
		}
	}))
	return <-done
}

// WritePacket writes blocks via scatter gather list.
func (bd *Bdev) WritePacket(blockOffset, blockCount int64, pkt pktmbuf.Packet) error {
	done := make(chan error)
	eal.PostMain(cptr.VoidFunction(func() {
		ctx := cptr.CtxPut(done)
		res := C.SpdkBdev_WritePacket(bd.c, bd.ch, (*C.struct_rte_mbuf)(pkt.Ptr()),
			C.uint64_t(blockOffset), C.uint64_t(blockCount), C.uint32_t(bd.blockSize),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- eal.Errno(-res)
			cptr.CtxClear(ctx)
		}
	}))
	return <-done
}

//export go_bdevIoComplete
func go_bdevIoComplete(io *C.struct_spdk_bdev_io, success C.bool, ctx unsafe.Pointer) {
	done := cptr.CtxPop(ctx).(chan error)
	if bool(success) {
		done <- nil
	} else {
		done <- errors.New("bdev_io error")
	}

	C.spdk_bdev_free_io(io)
}
