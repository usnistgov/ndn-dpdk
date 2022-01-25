package disk

/*
#include "../../csrc/disk/alloc.h"
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// Alloc is a disk slot allocator.
type Alloc struct {
	c *C.DiskAlloc
}

// Ptr returns *C.DiskAlloc pointer.
func (a *Alloc) Ptr() unsafe.Pointer {
	return unsafe.Pointer(a.c)
}

// Close destroys the Alloc.
func (a *Alloc) Close() error {
	eal.Free(a.c)
	return nil
}

// Alloc allocates a disk slot.
func (a *Alloc) Alloc() (slot uint64, e error) {
	s := C.DiskAlloc_Alloc(a.c)
	if s == 0 {
		return 0, errors.New("disk slot unavailable")
	}
	return uint64(s), nil
}

// Free frees a disk slot.
func (a *Alloc) Free(slot uint64) {
	C.DiskAlloc_Free(a.c, C.uint64_t(slot))
}

// NewAlloc creates an Alloc.
func NewAlloc(min, max uint64, socket eal.NumaSocket) *Alloc {
	a := C.DiskAlloc_New(C.uint64_t(min), C.uint64_t(max), C.int(socket.ID()))
	return &Alloc{c: a}
}
