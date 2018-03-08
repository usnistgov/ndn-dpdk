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
	StartDepth int
}

// The FIB.
type Fib struct {
	c          *C.Fib
	commands   chan command
	startDepth int
	nEntries   int
	nVirtuals  int
	tree       tree
}

type command interface {
	Execute(fib *Fib, rs *urcu.ReadSide)
}

func New(cfg Config) (fib *Fib, e error) {
	idC := C.CString(cfg.Id)
	defer C.free(unsafe.Pointer(idC))
	fib = new(Fib)
	fib.c = C.Fib_New(idC, C.uint32_t(cfg.MaxEntries), C.uint32_t(cfg.NBuckets),
		C.unsigned(cfg.NumaSocket), C.uint8_t(cfg.StartDepth))
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

	fib.startDepth = cfg.StartDepth
	return fib, nil
}

// Get native *C.Fib pointer to use in other packages.
func (fib *Fib) GetPtr() unsafe.Pointer {
	return unsafe.Pointer(fib.c)
}

// Get underlying mempool of the FIB.
func (fib *Fib) GetMempool() dpdk.Mempool {
	return dpdk.MempoolFromPtr(unsafe.Pointer(fib.c))
}

// Get number of FIB entries, excluding virtual entries.
func (fib *Fib) Len() int {
	return fib.nEntries
}

// Get number of virtual entries.
func (fib *Fib) CountVirtuals() int {
	return fib.nVirtuals
}

type closeCommand chan error

func (cmd closeCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	C.Fib_Close(fib.c)
	cmd <- nil
}

func (fib *Fib) Close() error {
	cmd := make(closeCommand)
	fib.commands <- cmd
	close(fib.commands)
	return <-cmd
}

type listNamesCommand chan []*ndn.Name

func (cmd listNamesCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	cmd <- fib.tree.List()
}

// List all FIB entry names.
func (fib *Fib) ListNames() []*ndn.Name {
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
	name := entry.GetName()
	comps := name.ListComps()
	nComps := len(comps)

	newEntryC := C.Fib_Alloc(fib.c)
	if newEntryC == nil {
		cmd.res <- errors.New("FIB entry allocation error")
		return
	}

	if nComps > fib.startDepth {
		virtNameV := ndn.JoinNameComponents(comps[:fib.startDepth])
		oldVirtC := fib.findC(virtNameV)
		if oldVirtC == nil || int(oldVirtC.maxDepth) < nComps-fib.startDepth {
			newVirtC := C.Fib_Alloc(fib.c)
			if newVirtC == nil {
				C.Fib_Free(fib.c, newEntryC)
				cmd.res <- errors.New("FIB virtual entry allocation error")
				return
			}
			if oldVirtC == nil {
				entrySetName(newVirtC, virtNameV, fib.startDepth)
				fib.nVirtuals++
			} else {
				*newVirtC = *oldVirtC
			}
			newVirtC.maxDepth = C.uint8_t(nComps - fib.startDepth)
			C.Fib_Insert(fib.c, newVirtC)
		}
	}

	*newEntryC = entry.c
	isReplacingVirtual := false
	if nComps == fib.startDepth {
		oldEntryC := fib.findC(name.GetValue())
		if oldEntryC != nil && oldEntryC.maxDepth > 0 {
			newEntryC.maxDepth = oldEntryC.maxDepth
			fib.nVirtuals--
			isReplacingVirtual = true
		}
	}

	if bool(C.Fib_Insert(fib.c, newEntryC)) || isReplacingVirtual {
		fib.nEntries++
		fib.tree.Insert(comps)
		cmd.res <- true
	} else {
		cmd.res <- false
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
	name *ndn.Name
	res  chan error
}

func (cmd eraseCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	rs.Lock()
	defer rs.Unlock()
	name := cmd.name
	comps := name.ListComps()
	nComps := len(comps)

	oldEntryC := fib.findC(name.GetValue())
	if oldEntryC == nil {
		cmd.res <- errors.New("FIB entry does not exist")
		return
	}
	oldMd, newMd := fib.tree.Erase(comps, fib.startDepth)

	var oldVirtC *C.FibEntry
	if nComps > fib.startDepth && oldMd != newMd {
		virtNameV := ndn.JoinNameComponents(comps[:fib.startDepth])
		oldVirtC = fib.findC(virtNameV)
	} else if nComps == fib.startDepth && newMd != 0 {
		oldVirtC = oldEntryC
		oldEntryC = nil // don't delete, because newVirtC is replacing oldEntryC
	}

	if oldVirtC != nil { // need to replace virtual entry
		newVirtC := C.Fib_Alloc(fib.c)
		if newVirtC == nil {
			fib.tree.Insert(comps) // revert tree change
			cmd.res <- errors.New("FIB virtual entry allocation error")
			return
		}

		*newVirtC = *oldVirtC
		newVirtC.maxDepth = C.uint8_t(newMd)
		C.Fib_Insert(fib.c, newVirtC)

		if newVirtC.nNexthops == 0 && oldMd == 0 && newMd > 0 {
			fib.nVirtuals++
		}

		// XXX if oldMd > 0 && newMd == 0, should delete and not replace virtual entry
	}

	fib.nEntries--
	if oldEntryC != nil {
		C.Fib_Erase(fib.c, oldEntryC)
	}
	cmd.res <- nil
}

// Erase a FIB entry by name.
func (fib *Fib) Erase(name *ndn.Name) error {
	cmd := eraseCommand{name: name, res: make(chan error)}
	fib.commands <- cmd
	return <-cmd.res
}

func (fib *Fib) findC(nameV ndn.TlvBytes) (entryC *C.FibEntry) {
	return C.__Fib_Find(fib.c, C.uint16_t(len(nameV)), (*C.uint8_t)(nameV.GetPtr()))
}

type findCommand struct {
	name *ndn.Name
	res  chan *Entry
}

func (cmd findCommand) Execute(fib *Fib, rs *urcu.ReadSide) {
	rs.Lock()
	defer rs.Unlock()
	name := cmd.name

	entryC := fib.findC(name.GetValue())
	if entryC == nil {
		cmd.res <- nil
	} else {
		cmd.res <- &Entry{*entryC}
	}
}

// Perform an exact match lookup.
// The FIB entry will be copied.
func (fib *Fib) Find(name *ndn.Name) (entry *Entry) {
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

	entryC := C.__Fib_Lpm(fib.c, (*C.PName)(name.GetPNamePtr()),
		(*C.uint8_t)(name.GetValue().GetPtr()))
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
