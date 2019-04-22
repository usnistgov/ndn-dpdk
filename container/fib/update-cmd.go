package fib

/*
#include "fib.h"
*/
import "C"
import (
	"errors"
	"fmt"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

// A sequence number on every inserted C.FibEntry, allowing to detect FIB changes.
var insertSeqNo uint32

func (fib *Fib) allocC(fibC *C.Fib) (entryC *C.FibEntry) {
	ok := bool(C.Fib_AllocBulk(fibC, &entryC, 1))
	if !ok {
		entryC = nil
	}
	return entryC
}

func (fib *Fib) insertC(fibC *C.Fib, entryC *C.FibEntry) (isNew bool) {
	insertSeqNo++
	entryC.seqNum = C.uint32_t(insertSeqNo)

	isNew = bool(C.Fib_Insert(fibC, entryC))
	if isNew {
		fib.nEntriesC++
	}
	return isNew
}

func (fib *Fib) eraseC(fibC *C.Fib, entryC *C.FibEntry) {
	entryC.shouldFreeDyn = true
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
		var dynMps []dpdk.Mempool
		var ndtIndex uint64
		if nComps < fib.ndt.GetPrefixLen() {
			logEntry = logEntry.WithField("partition", "all")
			fibsC = fib.c
			dynMps = fib.dynMps
		} else {
			var partition uint8
			ndtIndex, partition = fib.ndt.Lookup(name)
			logEntry = logEntry.WithFields(makeLogFields("partition", partition, "ndt-index", ndtIndex))
			if int(partition) >= len(fib.c) {
				return errors.New("bad partition")
			}
			fibsC = []*C.Fib{fib.c[partition]}
			dynMps = []dpdk.Mempool{fib.dynMps[partition]}
		}

		// allocate and populate new entries
		var newEntriesC []*C.FibEntry
		for i, fibC := range fibsC {
			newEntryC := fib.allocC(fibC)
			if newEntryC == nil {
				break
			}
			*newEntryC = entry.c
			newEntryC.dyn = (*C.FibEntryDyn)(dynMps[i].Alloc())
			if newEntryC.dyn == nil {
				C.Fib_Free(fibC, newEntryC)
				break
			}
			*newEntryC.dyn = C.FibEntryDyn{}
			newEntriesC = append(newEntriesC, newEntryC)
		}
		if len(newEntriesC) != len(fibsC) {
			for i, newEntryC := range newEntriesC {
				C.Fib_Free(fibsC[i], newEntryC)
			}
			return errors.New("allocation error")
		}

		// insert virtual entry if needed
		if name.Len() > fib.startDepth {
			// only one partition because cfg.StartDepth > ndt.GetPrefixLen()
			fibC := fibsC[0]
			virtNameV := ndn.JoinNameComponents(name.ListPrefixComps(fib.startDepth))
			virtNameHash := name.ComputePrefixHash(fib.startDepth)
			oldVirtC := findC(fibC, virtNameV, virtNameHash)
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
			oldEntryC := findC(fibsC[0], name.GetValue(), name.ComputeHash())
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
			if oldEntryC := findC(fibC, name.GetValue(), name.ComputeHash()); oldEntryC == nil {
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
				virtNameHash := name.ComputePrefixHash(fib.startDepth)
				oldVirtC := findC(fibC, virtNameV, virtNameHash) // is not nil
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
	newDynMp    dpdk.Mempool
	oldEntriesC []*C.FibEntry
	nOldDyns    int
	newEntriesC []*C.FibEntry
	newDynsC    []*C.FibEntryDyn

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
		ctx.newDynMp = fib.dynMps[newPartition]

		// find old entries
		for n, nameV := range fib.sti[ndtIndex] {
			nn := nodeName{NameV: string(nameV), NComps: fib.ndt.GetPrefixLen()}
			n.Walk(nn, func(nn nodeName, node *node) {
				name := nn.GetName()
				if node.IsEntry || (node.MaxDepth > 0 && nn.NComps == fib.startDepth) {
					oldEntryC := findC(ctx.oldFibC, name.GetValue(), name.ComputeHash())
					if oldEntryC == nil {
						panic(fmt.Sprintf("entry not found %s", name))
					}
					ctx.oldEntriesC = append(ctx.oldEntriesC, oldEntryC)
					if oldEntryC.dyn != nil {
						ctx.nOldDyns++
					}
				}
			})
		}
		logEntry = logEntry.WithFields(makeLogFields("nEntries", len(ctx.oldEntriesC),
			"nSubtrees", len(fib.sti[ndtIndex])))

		// allocate new dyns
		if ctx.nOldDyns > 0 {
			ctx.newDynsC = make([]*C.FibEntryDyn, ctx.nOldDyns)
			if e := ctx.newDynMp.AllocBulk(ctx.newDynsC); e != nil {
				return e
			}
		}

		// allocate new entries
		if len(ctx.oldEntriesC) > 0 {
			ctx.newEntriesC = make([]*C.FibEntry, len(ctx.oldEntriesC))
			if ok := bool(C.Fib_AllocBulk(ctx.newFibC, &ctx.newEntriesC[0], C.unsigned(len(ctx.newEntriesC)))); !ok {
				ctx.newDynMp.FreeBulk(ctx.newDynsC)
				return errors.New("allocation error")
			}
		}

		// insert new entries
		j := 0
		for i, oldEntryC := range ctx.oldEntriesC {
			newEntryC := ctx.newEntriesC[i]
			*newEntryC = *oldEntryC
			if oldEntryC.dyn != nil {
				newEntryC.dyn = ctx.newDynsC[j]
				j++
				C.FibEntryDyn_Copy(newEntryC.dyn, oldEntryC.dyn)
			}
			if isNew := fib.insertC(ctx.newFibC, newEntryC); !isNew {
				newEntry := Entry{*oldEntryC}
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
