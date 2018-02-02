package fib

/*
#include "fib.h"
*/
import "C"
import (
	"errors"
	"unsafe"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

type Config struct {
	Id         string
	MaxEntries int
	NBuckets   int
	NumaSocket dpdk.NumaSocket
}

// The FIB.
type Fib struct {
	c        *C.Fib
	nEntries int
}

func New(cfg Config) (fib *Fib, e error) {
	idC := C.CString(cfg.Id)
	defer C.free(unsafe.Pointer(idC))
	fib = new(Fib)
	fib.c = C.Fib_New(idC, C.uint32_t(cfg.MaxEntries), C.uint32_t(cfg.NBuckets),
		C.unsigned(cfg.NumaSocket))

	if fib.c == nil {
		return nil, dpdk.GetErrno()
	}
	return fib, nil
}

func (fib *Fib) Close() error {
	C.Fib_Close(fib.c)
	return nil
}

// Get underlying mempool of the FIB.
func (fib *Fib) GetMempool() dpdk.Mempool {
	return dpdk.MempoolFromPtr(unsafe.Pointer(fib.c))
}

// Get number of FIB entries.
func (fib *Fib) Len() int {
	return fib.nEntries
}

// Insert a FIB entry.
// If an existing entry has the same name, it will be replaced.
func (fib *Fib) Insert(entry *Entry) (isNew bool, e error) {
	if entry.c.nNexthops == 0 {
		return false, errors.New("cannot insert FIB entry with no nexthop")
	}

	res := C.Fib_Insert(fib.c, &entry.c)
	switch res {
	case C.FIB_INSERT_REPLACE:
		return false, nil
	case C.FIB_INSERT_NEW:
		fib.nEntries++
		return true, nil
	case C.FIB_INSERT_ALLOC_ERROR:
		return false, errors.New("FIB entry allocation error")
	}
	panic("C.Fib_Insert unexpected return value")
}

// Erase a FIB entry by name.
func (fib *Fib) Erase(name ndn.TlvBytes) (ok bool) {
	ok = bool(C.Fib_Erase(fib.c, C.uint16_t(len(name)), (*C.uint8_t)(name.GetPtr())))
	if ok {
		fib.nEntries--
	}
	return ok
}

// Perform a longest prefix match lookup.
// The FIB entry will be copied.
func (fib *Fib) Lpm(name *ndn.Name, rcuRs *urcu.ReadSide) (entry *Entry) {
	rcuRs.Lock()
	defer rcuRs.Unlock()

	entryC := C.Fib_Lpm(fib.c, (*C.Name)(name.GetPtr()))
	if entryC == nil {
		return nil
	}

	entry = new(Entry)
	entry.c = *entryC
	return entry
}