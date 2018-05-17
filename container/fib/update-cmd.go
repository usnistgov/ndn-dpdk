package fib

/*
#include "fib.h"
*/
import "C"
import (
	"errors"
	"fmt"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/ndn"
)

func (fib *Fib) allocC(fibC *C.Fib) (entryC *C.FibEntry) {
	ok := bool(C.Fib_AllocBulk(fibC, &entryC, 1))
	if !ok {
		entryC = nil
	}
	return entryC
}

func (fib *Fib) insertC(fibC *C.Fib, entryC *C.FibEntry) (isNew bool) {
	isNew = bool(C.Fib_Insert(fibC, entryC))
	if isNew {
		fib.nEntriesC++
	}
	return isNew
}

func (fib *Fib) eraseC(fibC *C.Fib, entryC *C.FibEntry) {
	C.Fib_Erase(fibC, entryC)
	fib.nEntriesC--
}

// Insert a FIB entry.
// If an existing entry has the same name, it will be replaced.
func (fib *Fib) Insert(entry *Entry) (isNew bool, e error) {
	if entry.c.nNexthops == 0 {
		return false, errors.New("cannot insert FIB entry with no nexthop")
	}
	if entry.c.strategy == nil {
		return false, errors.New("cannot insert FIB entry without strategy")
	}
	name := entry.GetName()
	nComps := name.Len()
	logEntry := log.WithFields(makeLogFields("name", name, "nexthops", entry.GetNexthops(), "strategy", entry.GetStrategy().GetId()))

	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()

		// determine what partition(s) should receive new entry
		var fibsC []*C.Fib
		var ndtIndex uint64
		if nComps < fib.ndt.GetPrefixLen() {
			logEntry = logEntry.WithField("partition", "all")
			fibsC = fib.c
		} else {
			var partition uint8
			ndtIndex, partition = fib.ndt.Lookup(name)
			logEntry = logEntry.WithFields(makeLogFields("partition", partition, "ndt-index", ndtIndex))
			if int(partition) >= len(fib.c) {
				return errors.New("bad partition")
			}
			fibsC = []*C.Fib{fib.c[partition]}
		}

		// allocate and populate new entries
		var newEntriesC []*C.FibEntry
		for _, fibC := range fibsC {
			if newEntryC := fib.allocC(fibC); newEntryC == nil {
				for i, allocatedEntryC := range newEntriesC {
					C.Fib_Free(fibsC[i], allocatedEntryC)
				}
				return errors.New("allocation error")
			} else {
				*newEntryC = entry.c
				newEntriesC = append(newEntriesC, newEntryC)
			}
		}

		// insert virtual entry if needed
		if name.Len() > fib.startDepth {
			// only one partition because cfg.StartDepth > ndt.GetPrefixLen()
			fibC := fibsC[0]
			virtNameV := ndn.JoinNameComponents(name.ListPrefixComps(fib.startDepth))
			oldVirtC := findC(fibC, virtNameV)
			if oldVirtC == nil || int(oldVirtC.maxDepth) < nComps-fib.startDepth {
				newVirtC := fib.allocC(fibC)
				if newVirtC == nil {
					C.Fib_Free(fibC, newEntriesC[0])
					return errors.New("allocation error")
				}
				if oldVirtC == nil {
					*newVirtC = C.FibEntry{}
					entrySetName(newVirtC, virtNameV, fib.startDepth)
				} else {
					*newVirtC = *oldVirtC
				}
				newVirtC.maxDepth = C.uint8_t(nComps - fib.startDepth)
				fib.insertC(fibC, newVirtC)
			}
		}

		// if there was a virtual entry at the same place as the new entry, copy its maxDepth
		isReplacingVirtual := false
		if nComps == fib.startDepth {
			// only one partition because cfg.StartDepth > ndt.GetPrefixLen()
			oldEntryC := findC(fibsC[0], name.GetValue())
			if oldEntryC != nil && oldEntryC.maxDepth > 0 {
				newEntriesC[0].maxDepth = oldEntryC.maxDepth
				isReplacingVirtual = true
			}
		}

		// insert new entries
		for i, newEntryC := range newEntriesC {
			isNew = fib.insertC(fibsC[i], newEntryC) || isReplacingVirtual
		}
		if isNew {
			fib.insertNode(name, ndtIndex)
		}
		return nil
	})

	if e != nil {
		logEntry.WithError(e).Error("Insert")
	} else {
		logEntry.Info("Insert")
	}
	return isNew, e
}

// Erase a FIB entry by name.
func (fib *Fib) Erase(name *ndn.Name) (e error) {
	nComps := name.Len()
	logEntry := log.WithField("name", name)

	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()

		// determine what partition(s) are affected
		var fibsC []*C.Fib
		var ndtIndex uint64
		if nComps < fib.ndt.GetPrefixLen() {
			logEntry = logEntry.WithField("partition", "all")
			fibsC = fib.c
		} else {
			var partition uint8
			ndtIndex, partition = fib.ndt.Lookup(name)
			logEntry = logEntry.WithFields(makeLogFields("partition", partition, "ndt-index", ndtIndex))
			if int(partition) >= len(fib.c) {
				return errors.New("bad partition")
			}
			fibsC = []*C.Fib{fib.c[partition]}
		}

		// retrieve old entries
		var oldEntriesC []*C.FibEntry
		for _, fibC := range fibsC {
			if oldEntryC := findC(fibC, name.GetValue()); oldEntryC == nil {
				return errors.New("entry does not exist")
			} else {
				oldEntriesC = append(oldEntriesC, oldEntryC)
			}
		}

		// update tree
		oldMd, newMd := fib.eraseNode(name, ndtIndex)
		success := false
		defer func() {
			if !success {
				fib.insertNode(name, ndtIndex)
			}
		}()

		if nComps >= fib.startDepth {
			// only one partition because cfg.StartDepth > ndt.GetPrefixLen()
			fibC := fibsC[0]

			if nComps > fib.startDepth && oldMd != newMd {
				virtNameV := ndn.JoinNameComponents(name.ListPrefixComps(fib.startDepth))
				oldVirtC := findC(fibC, virtNameV) // is not nil
				if newMd == 0 {
					// erase virtual entry
					fib.eraseC(fibC, oldVirtC)
				} else {
					// update virtual entry
					newVirtC := fib.allocC(fibC)
					if newVirtC == nil {
						return errors.New("allocation error")
					}
					*newVirtC = *oldVirtC
					newVirtC.maxDepth = C.uint8_t(newMd)
					fib.insertC(fibC, newVirtC)
				}
			} else if nComps == fib.startDepth && newMd != 0 {
				// replace oldEntriesC[0] with virtual entry
				newVirtC := fib.allocC(fibC)
				if newVirtC == nil {
					return errors.New("allocation error")
				}
				*newVirtC = C.FibEntry{}
				entrySetName(newVirtC, name.GetValue(), nComps)
				newVirtC.maxDepth = C.uint8_t(newMd)
				fib.insertC(fibC, newVirtC)
				oldEntriesC = nil // don't delete oldEntriesC[0]
			}
		}

		// erase old entries
		for i, oldEntryC := range oldEntriesC {
			fib.eraseC(fibsC[i], oldEntryC)
		}
		success = true
		return nil
	})

	if e != nil {
		logEntry.WithError(e).Error("Erase")
	} else {
		logEntry.Info("Erase")
	}
	return e
}

// Context of relocate operation.
type RelocateContext struct {
	oldFibC     *C.Fib
	newFibC     *C.Fib
	oldEntriesC []*C.FibEntry
	newEntriesC []*C.FibEntry

	NoRevertOnError bool // If true, relocating is not reverted even if callback has error.
}

// Get how many FIB entries are being moved.
func (ctx *RelocateContext) Len() int {
	return len(ctx.oldEntriesC)
}

// Callback during relocate operation.
// It is invoked after entries are inserted to new partition, but before entries are erased
// from old partition. The callback should perform NDT update, then sleep long enough for
// previous dispatched packets that depend on old entries to be processed. Note that the FIB
// could not process other commands during this sleep period. In case the callback errors,
// relocating operation will be reverted, unless ctx.NoRevertOnError is set to true.
type RelocateCallback func(ctx *RelocateContext) error

// Relocate entries under an NDT index from one partition to another.
func (fib *Fib) Relocate(ndtIndex uint64, oldPartition, newPartition uint8,
	cb RelocateCallback) (e error) {
	logEntry := log.WithFields(makeLogFields("ndtIndex", ndtIndex,
		"oldPartition", oldPartition, "newPartition", newPartition))
	if oldPartition == newPartition {
		logEntry.Info("Relocate noop")
		return nil
	}

	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		if int(oldPartition) >= len(fib.c) {
			return errors.New("bad old partition")
		}
		if int(newPartition) >= len(fib.c) {
			return errors.New("bad new partition")
		}

		rs.Lock()
		defer rs.Unlock()

		var ctx RelocateContext
		ctx.oldFibC = fib.c[oldPartition]
		ctx.newFibC = fib.c[newPartition]

		// find old entries
		for n, nameV := range fib.sti[ndtIndex] {
			nn := nodeName{NameV: string(nameV), NComps: fib.ndt.GetPrefixLen()}
			n.Walk(nn, func(nn nodeName, node *node) {
				if node.IsEntry || (node.MaxDepth > 0 && nn.NComps == fib.startDepth) {
					if oldEntryC := findC(ctx.oldFibC, ndn.TlvBytes(nn.NameV)); oldEntryC == nil {
						panic(fmt.Sprintf("entry not found %s", nn.GetName()))
					} else {
						ctx.oldEntriesC = append(ctx.oldEntriesC, oldEntryC)
					}
				}
			})
		}
		logEntry = logEntry.WithFields(makeLogFields("nEntries", len(ctx.oldEntriesC),
			"nSubtrees", len(fib.sti[ndtIndex])))

		// allocate new entries
		if len(ctx.oldEntriesC) > 0 {
			ctx.newEntriesC = make([]*C.FibEntry, len(ctx.oldEntriesC))
			if ok := bool(C.Fib_AllocBulk(ctx.newFibC, &ctx.newEntriesC[0], C.unsigned(len(ctx.newEntriesC)))); !ok {
				return errors.New("allocation error")
			}
		}

		// insert new entries
		for i, oldEntryC := range ctx.oldEntriesC {
			newEntryC := ctx.newEntriesC[i]
			*newEntryC = *oldEntryC
			if isNew := fib.insertC(ctx.newFibC, newEntryC); !isNew {
				newEntry := Entry{*newEntryC}
				panic(fmt.Sprintf("entry should not exist %s", newEntry.GetName()))
			}
			fib.nRelocatingEntriesC++
		}

		// invoke callback
		cbErr := cb(&ctx)
		if cbErr == nil || ctx.NoRevertOnError {
			// erase old entries
			for _, oldEntryC := range ctx.oldEntriesC {
				fib.eraseC(ctx.oldFibC, oldEntryC)
				fib.nRelocatingEntriesC--
			}
		} else {
			// revert: erase new entries
			for _, newEntryC := range ctx.newEntriesC {
				fib.eraseC(ctx.newFibC, newEntryC)
				fib.nRelocatingEntriesC--
			}
		}

		return cbErr
	})

	if e != nil {
		logEntry.WithError(e).Error("Relocate")
	} else {
		logEntry.Info("Relocate")
	}
	return e
}
