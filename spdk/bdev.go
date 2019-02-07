package spdk

/*
#include <rte_memcpy.h>
#include <spdk/bdev.h>
#include <spdk/thread.h>

extern void go_bdevInit(void* ctx, int rc);
extern void go_bdevIoComplete(struct spdk_bdev_io* io, bool success, void* ctx);
*/
import "C"
import (
	"errors"
	"io"
	"unsafe"

	"ndn-dpdk/dpdk"
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
		if res := C.spdk_bdev_open(bdi.c, C.bool(mode), nil, nil, &bd.c); res != 0 {
			e = dpdk.Errno(res)
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

// Close the block device.
func (bd *Bdev) Close() error {
	MainThread.Call(func() {
		C.spdk_put_io_channel(bd.ch)
		C.spdk_bdev_close(bd.c)
	})
	return nil
}

// Obtain BdevInfo.
func (bd *Bdev) GetInfo() (bdi BdevInfo) {
	bdi.c = C.spdk_bdev_desc_get_bdev(bd.c)
	return bdi
}

// Read from block device.
func (bd *Bdev) ReadBlocks(blockOffset, blockCount int64, buf []byte) error {
	if blockOffset < 0 || blockOffset+blockCount >= bd.nBlocks {
		return io.ErrUnexpectedEOF
	}
	sizeofBuf := blockCount * bd.blockSize
	if sizeofBuf > int64(len(buf)) {
		return io.ErrShortBuffer
	}

	bufC := dpdk.Zmalloc("SpdkBdevBuf", sizeofBuf, dpdk.NUMA_SOCKET_ANY)
	defer dpdk.Free(bufC)

	done := make(chan error)
	MainThread.Post(func() {
		ctx := ctxPut(done)
		res := C.spdk_bdev_read_blocks(bd.c, bd.ch, bufC, C.uint64_t(blockOffset), C.uint64_t(blockCount),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- dpdk.Errno(-res)
			ctxClear(ctx)
		}
	})
	if e := <-done; e != nil {
		return e
	}

	C.rte_memcpy(unsafe.Pointer(&buf[0]), bufC, C.size_t(sizeofBuf))
	return nil
}

// Write to block device.
func (bd *Bdev) WriteBlocks(blockOffset, blockCount int64, buf []byte) error {
	if blockOffset < 0 || blockOffset+blockCount >= bd.nBlocks {
		return io.ErrShortWrite
	}
	sizeofBuf := blockCount * bd.blockSize
	if sizeofBuf > int64(len(buf)) {
		return io.ErrShortBuffer
	}

	bufC := dpdk.Zmalloc("SpdkBdevBuf", sizeofBuf, dpdk.NUMA_SOCKET_ANY)
	defer dpdk.Free(bufC)
	C.rte_memcpy(bufC, unsafe.Pointer(&buf[0]), C.size_t(sizeofBuf))

	done := make(chan error)
	MainThread.Post(func() {
		ctx := ctxPut(done)
		res := C.spdk_bdev_write_blocks(bd.c, bd.ch, bufC, C.uint64_t(blockOffset), C.uint64_t(blockCount),
			C.spdk_bdev_io_completion_cb(C.go_bdevIoComplete), ctx)
		if res != 0 {
			done <- dpdk.Errno(-res)
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
