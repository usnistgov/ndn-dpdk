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
	commands chan command
	nEntries int
	tree     tree
}

type command interface {
	Execute(fib *Fib, rs *urcu.ReadSide)
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
	fib.commands = make(chan command)

	go func() {
		// execute all commands in a RCU read-side thread
		rs := urcu.NewReadSide()
		defer rs.Close()
		rs.Offline()
		for cmd, ok := <-fib.commands; ok; cmd, ok = <-fib.commands {
			rs.Online()
			cmd.Execute(fib, rs)
			rs.Offline()
		}
	}()
	return fib, nil
}

type closeCommand chan error

func (cmd closeCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	C.Fib_Close(fib.c)
	cmd <- nil
}

func (fib *Fib) Close() error {
	cmd := make(closeCommand)
	fib.commands <- cmd
	return <-cmd
}

// Get underlying mempool of the FIB.
func (fib *Fib) GetMempool() dpdk.Mempool {
	return dpdk.MempoolFromPtr(unsafe.Pointer(fib.c))
}

// Get number of FIB entries.
func (fib *Fib) Len() int {
	return fib.nEntries
}

type listNamesCommand chan []ndn.TlvBytes

func (cmd listNamesCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	cmd <- fib.tree.List()
}

// List all FIB entry names.
func (fib *Fib) ListNames() []ndn.TlvBytes {
	cmd := make(listNamesCommand)
	fib.commands <- cmd
	return <-cmd
}

type insertCommand struct {
	entry *Entry
	res   chan interface{}
}

func (cmd insertCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	rs.Lock()
	defer rs.Unlock()

	entry := cmd.entry
	res := C.Fib_Insert(fib.c, &entry.c)
	switch res {
	case C.FIB_INSERT_REPLACE:
		cmd.res <- false
	case C.FIB_INSERT_NEW:
		fib.nEntries++
		fib.tree.Insert(entry.GetName())
		cmd.res <- true
	case C.FIB_INSERT_ALLOC_ERROR:
		cmd.res <- errors.New("FIB entry allocation error")
	default:
		panic("C.Fib_Insert unexpected return value")
	}
}

// Insert a FIB entry.
// If an existing entry has the same name, it will be replaced.
func (fib *Fib) Insert(entry *Entry) (isNew bool, e error) {
	if entry.c.nNexthops == 0 {
		return false, errors.New("cannot insert FIB entry with no nexthop")
	}

	cmd := insertCommand{entry: entry, res: make(chan interface{})}
	fib.commands <- cmd
	switch res := (<-cmd.res).(type) {
	case bool:
		return res, nil
	case error:
		return false, res
	}
	panic(nil)
}

type eraseCommand struct {
	name ndn.TlvBytes
	res  chan bool
}

func (cmd eraseCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	rs.Lock()
	defer rs.Unlock()

	name := cmd.name
	ok := bool(C.Fib_Erase(fib.c, C.uint16_t(len(name)), (*C.uint8_t)(name.GetPtr())))
	if ok {
		fib.nEntries--
		fib.tree.Erase(name)
	}
	cmd.res <- ok
}

// Erase a FIB entry by name.
func (fib *Fib) Erase(name ndn.TlvBytes) (ok bool) {
	cmd := eraseCommand{name: name, res: make(chan bool)}
	fib.commands <- cmd
	return <-cmd.res
}

type findCommand struct {
	name ndn.TlvBytes
	res  chan *Entry
}

func (cmd findCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	rs.Lock()
	defer rs.Unlock()

	name := cmd.name
	entryC := C.Fib_Find(fib.c, C.uint16_t(len(name)), (*C.uint8_t)(name.GetPtr()))
	if entryC == nil {
		cmd.res <- nil
	} else {
		cmd.res <- &Entry{*entryC}
	}
}

// Perform an exact match lookup.
// The FIB entry will be copied.
func (fib *Fib) Find(name ndn.TlvBytes) (entry *Entry) {
	cmd := findCommand{name: name, res: make(chan *Entry)}
	fib.commands <- cmd
	return <-cmd.res
}

type lpmCommand struct {
	name *ndn.Name
	res  chan *Entry
}

func (cmd lpmCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	rs.Lock()
	defer rs.Unlock()

	name := cmd.name
	entryC := C.Fib_Lpm(fib.c, (*C.Name)(name.GetPtr()))
	if entryC == nil {
		cmd.res <- nil
	} else {
		cmd.res <- &Entry{*entryC}
	}
}

// Perform a longest prefix match lookup.
// The FIB entry will be copied.
func (fib *Fib) Lpm(name *ndn.Name) (entry *Entry) {
	cmd := lpmCommand{name: name, res: make(chan *Entry)}
	fib.commands <- cmd
	return <-cmd.res
}
