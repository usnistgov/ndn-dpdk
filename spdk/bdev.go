package spdk

/*
#include "bdev.h"
#include <spdk/thread.h>

extern void go_bdevEvent(enum spdk_bdev_event_type type, struct spdk_bdev* bdev, void* ctx);
extern void go_bdevIoComplete(struct spdk_bdev_io* io, bool success, void* ctx);
*/
import "C"
import (
	"errors"
	"io"
	"unsafe"

	"ndn-dpdk/dpdk/eal"
	"ndn-dpdk/dpdk/pktmbuf"
)

// Mode of opening a block device.
type BdevMode bool

const (
	BDEV_MODE_READ_ONLY  BdevMode = false
	BDEV_MODE_READ_WRITE BdevMode = true
)

// Open block device descriptor.
type Bdev struct {
	c         *C.struct_spdk_bdev_desc
	ch        *C.struct_spdk_io_channel
	blockSize int64
	nBlocks   int64
}

// Open a block device.
func OpenBdev(bdi BdevInfo, mode BdevMode) (bd *Bdev, e error) {
	bd = new(Bdev)
	MainThread.Call(func() {
		if res := C.spdk_bdev_open_ext(C.spdk_bdev_get_name(bdi.c), C.bool(mode),
			C.spdk_bdev_event_cb_t(C.go_bdevEvent), nil, &bd.c); res != 0 {
			e = eal.Errno(res)
			return
		}
		bd.ch = C.spdk_bdev_get_io_channel(bd.c)
	})
	if e != nil {
		return nil, e
	}
	bd.blockSize = int64(bdi.GetBlockSize())
	bd.nBlocks = int64(bdi.CountBlocks())
	return bd, nil
}

//export go_bdevEvent
func go_bdevEvent(typ C.enum_spdk_bdev_event_type, bdev *C.struct_spdk_bdev, ctx unsafe.Pointer) {
}

// Close the block device.
func (bd *Bdev) Close() error {
	MainThread.Call(func() {
		C.spdk_put_io_channel(bd.ch)
		C.spdk_bdev_close(bd.c)
	})
	return nil
}

// Get native *C.struct_spdk_bdev_desc pointer to use in other packages.
func (bd *Bdev) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(bd.c)
}

// Obtain BdevInfo.
func (bd *Bdev) GetInfo() (bdi BdevInfo) {
	bdi.c = C.spdk_bdev_desc_get_bdev(bd.c)
	return bdi
}

// Read blocks.
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
	MainThread.Post(func() {
		ctx := ctxPut(done)
		res := C.spdk_bdev_read_blocks(bd.c, bd.ch, bufC, C.uint64_t(blockOffset), C.uint64_t(blockCount),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- eal.Errno(-res)
			ctxClear(ctx)
		}
	})
	if e := <-done; e != nil {
		return e
	}

	C.rte_memcpy(unsafe.Pointer(&buf[0]), bufC, C.size_t(sizeofBuf))
	return nil
}

// Write blocks.
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
	MainThread.Post(func() {
		ctx := ctxPut(done)
		res := C.spdk_bdev_write_blocks(bd.c, bd.ch, bufC, C.uint64_t(blockOffset), C.uint64_t(blockCount),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- eal.Errno(-res)
			ctxClear(ctx)
		}
	})
	return <-done
}

// Read blocks via scatter gather list.
func (bd *Bdev) ReadPacket(blockOffset, blockCount int64, pkt pktmbuf.Packet) error {
	done := make(chan error)
	MainThread.Post(func() {
		ctx := ctxPut(done)
		res := C.SpdkBdev_ReadPacket(bd.c, bd.ch, (*C.struct_rte_mbuf)(pkt.GetPtr()),
			C.uint64_t(blockOffset), C.uint64_t(blockCount), C.uint32_t(bd.blockSize),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- eal.Errno(-res)
			ctxClear(ctx)
		}
	})
	return <-done
}

// Write blocks via scatter gather list.
func (bd *Bdev) WritePacket(blockOffset, blockCount int64, pkt pktmbuf.Packet) error {
	done := make(chan error)
	MainThread.Post(func() {
		ctx := ctxPut(done)
		res := C.SpdkBdev_WritePacket(bd.c, bd.ch, (*C.struct_rte_mbuf)(pkt.GetPtr()),
			C.uint64_t(blockOffset), C.uint64_t(blockCount), C.uint32_t(bd.blockSize),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- eal.Errno(-res)
			ctxClear(ctx)
		}
	})
	return <-done
}

//export go_bdevIoComplete
func go_bdevIoComplete(io *C.struct_spdk_bdev_io, success C.bool, ctx unsafe.Pointer) {
	done := ctxPop(ctx).(chan error)
	if bool(success) {
		done <- nil
	} else {
		done <- errors.New("bdev_io error")
	}

	C.spdk_bdev_free_io(io)
}

// Read bytes at specific offset.
func (bd *Bdev) ReadAt(p []byte, off int64) (n int, e error) {
	if len(p) == 0 {
		return 0, nil
	}

	blockOffset := off / bd.blockSize
	lastByteOffset := off + int64(len(p))
	lastBlockOffset := lastByteOffset / bd.blockSize
	if lastByteOffset%bd.blockSize > 0 {
		lastBlockOffset++
	}
	blockCount := lastBlockOffset - blockOffset + 1

	if off%bd.blockSize == 0 {
		if e := bd.ReadBlocks(blockOffset, blockCount, p); e != nil {
			return 0, e
		}
		return len(p), nil
	}

	buf := make([]byte, int(blockCount*bd.blockSize))
	if e := bd.ReadBlocks(blockOffset, blockCount, buf); e != nil {
		return 0, e
	}
	return copy(p, buf[off%bd.blockSize:]), nil
}

// Write bytes at specific offset.
// Since bdev can only write whole blocks, other bytes in affected blocks will be zeroed.
func (bd *Bdev) WriteAt(p []byte, off int64) (n int, e error) {
	if len(p) == 0 {
		return 0, nil
	}

	blockOffset := off / bd.blockSize
	lastByteOffset := off + int64(len(p))
	lastBlockOffset := lastByteOffset / bd.blockSize
	if lastByteOffset%bd.blockSize > 0 {
		lastBlockOffset++
	}
	blockCount := lastBlockOffset - blockOffset + 1

	if off%bd.blockSize != 0 {
		buf := make([]byte, int(blockCount*bd.blockSize))
		copy(buf[off%bd.blockSize:], p)
		p = buf
	}

	if e := bd.WriteBlocks(blockOffset, blockCount, p); e != nil {
		return 0, e
	}
	return len(p), nil
}
