package fib

/*
#include "fib.h"
*/
import "C"
import (
	"errors"
	"fmt"

	"ndn-dpdk/container/fib/fibtree"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/ndn"
)

type updateAct int

const (
	updateActInsert updateAct = 1
	updateActErase  updateAct = 2
)

type updateItem struct {
	act   updateAct
	part  *partition
	entry *C.FibEntry
}

type updateBatch []updateItem

func (batch updateBatch) Apply() {
	for _, item := range batch {
		if item.act == updateActInsert {
			item.part.Insert(item.entry)
		} else if item.act == updateActErase {
			item.part.Erase(item.entry)
		}
	}
}

func (batch updateBatch) Discard(part *partition) error {
	for _, item := range batch {
		if item.act == updateActInsert {
			C.Fib_Free(item.part.c, item.entry)
		}
	}
	return fmt.Errorf("allocation error in partition %d", part.index)
}

func (fib *Fib) getVirtName(name *ndn.Name) *ndn.Name {
	if name.Len() < fib.cfg.StartDepth {
		return nil
	}
	return name.GetPrefix(fib.cfg.StartDepth)
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
	virtName := fib.getVirtName(name)
	logEntry := log.WithFields(makeLogFields("name", name, "nexthops", entry.GetNexthops(), "strategy", entry.GetStrategy().GetId()))

	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()

		// update tree
		isNewInTree, oldMd, newMd, virtIsEntry := fib.tree.Insert(name)
		success := false
		defer func() {
			if !success && isNewInTree {
				fib.tree.Erase(name)
			}
		}()
		logEntry = logEntry.WithField("isNew", isNewInTree)
		isNew = isNewInTree

		// determine what partition(s) should receive new entry
		parts := fib.listPartitionsForName(name)
		logEntry = logEntry.WithField("partition", listPartitionNumbers(parts))

		var batch updateBatch
		for _, part := range parts {
			// prepare new entry
			newEntry := part.Alloc()
			if newEntry == nil {
				return batch.Discard(part)
			}
			*newEntry = entry.c
			batch = append(batch, updateItem{updateActInsert, part, newEntry})
			if dyn := (*C.FibEntryDyn)(part.dynMp.Alloc()); dyn == nil {
				return batch.Discard(part)
			} else {
				newEntry.dyn = dyn
			}

			switch {
			case name.Len() < fib.cfg.StartDepth:
				// virtual entry not involved

			case name.Len() == fib.cfg.StartDepth:
				// inherit maxDepth; it would be zero if there wasn't a virtual entry at name
				newEntry.maxDepth = C.uint8_t(newMd)

			case name.Len() > fib.cfg.StartDepth && oldMd == newMd:
				// no virtual entry update necessary

			case name.Len() > fib.cfg.StartDepth && virtIsEntry:
				// update maxDepth on entry at virtName
				oldVirt := part.Find(virtName)
				if oldVirt == nil {
					panic(fmt.Errorf("entry %s missing in partition %d", virtName, part.index))
				}
				newVirt := part.Alloc()
				if newVirt == nil {
					return batch.Discard(part)
				}
				*newVirt = *oldVirt
				newVirt.maxDepth = C.uint8_t(newMd)
				batch = append(batch, updateItem{updateActInsert, part, newVirt})

			case name.Len() > fib.cfg.StartDepth && !virtIsEntry:
				// insert or update virtual entry
				newVirt := part.Alloc()
				if newVirt == nil {
					return batch.Discard(part)
				}
				entrySetName(newVirt, virtName)
				newVirt.maxDepth = C.uint8_t(newMd)
				batch = append(batch, updateItem{updateActInsert, part, newVirt})

			default:
				panic("unexpected case")
			}
		}

		// perform batch updates
		batch.Apply()
		success = true
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
	virtName := fib.getVirtName(name)
	logEntry := log.WithField("name", name)

	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()

		// update tree
		isErasedInTree, oldMd, newMd, virtIsEntry := fib.tree.Erase(name)
		if !isErasedInTree {
			logEntry = logEntry.WithField("skip", "no-entry")
			return errors.New("entry does not exist")
		}
		success := false
		defer func() {
			if !success {
				fib.tree.Insert(name)
			}
		}()

		// determine what partition(s) are affected
		parts := fib.listPartitionsForName(name)
		logEntry = logEntry.WithField("partition", listPartitionNumbers(parts))

		var batch updateBatch
		for _, part := range parts {
			// retrieve old entry
			oldEntry := part.Find(name)
			if oldEntry == nil {
				panic(fmt.Errorf("entry %s missing in partition %d", name, part.index))
			}

			switch {
			case name.Len() < fib.cfg.StartDepth:
				// virtual entry not involved

			case name.Len() == fib.cfg.StartDepth && newMd == 0:
				// erase old entry without replacing with virtual entry

			case name.Len() == fib.cfg.StartDepth && newMd > 0:
				// replace old entry with virtual entry
				newVirt := part.Alloc()
				if newVirt == nil {
					return batch.Discard(part)
				}
				entrySetName(newVirt, virtName)
				newVirt.maxDepth = C.uint8_t(newMd)
				batch = append(batch, updateItem{updateActInsert, part, newVirt})
				oldEntry = nil // don't erase oldEntry: it will be replaced by newVirt
				// XXX is dyn released correctly?

			case name.Len() > fib.cfg.StartDepth && oldMd == newMd:
				// no virtual entry update necessary

			case name.Len() > fib.cfg.StartDepth && virtIsEntry:
				// update maxDepth on entry at virtName
				oldVirt := part.Find(virtName)
				if oldVirt == nil {
					panic(fmt.Errorf("virtual entry %s missing in partition %d", virtName, part.index))
				}
				newVirt := part.Alloc()
				if newVirt == nil {
					return batch.Discard(part)
				}
				*newVirt = *oldVirt // copy nexthops/strategy/dyn/etc
				newVirt.maxDepth = C.uint8_t(newMd)
				batch = append(batch, updateItem{updateActInsert, part, newVirt})

			case name.Len() > fib.cfg.StartDepth && !virtIsEntry:
				oldVirt := part.Find(virtName)
				if oldVirt == nil {
					panic(fmt.Errorf("virtual entry %s missing in partition %d", virtName, part.index))
				}
				if newMd == 0 {
					// erase virtual entry
					batch = append(batch, updateItem{updateActErase, part, oldVirt})
				} else {
					// update virtual entry
					newVirt := part.Alloc()
					if newVirt == nil {
						return batch.Discard(part)
					}
					newVirt.maxDepth = C.uint8_t(newMd)
					batch = append(batch, updateItem{updateActInsert, part, newVirt})
				}

			default:
				panic("unexpected case")
			}

			if oldEntry != nil {
				batch = append(batch, updateItem{updateActErase, part, oldEntry})
			}
		}

		// perform batch updates
		batch.Apply()
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

// Callback during relocate operation.
// It is invoked after entries are inserted to new partition, but before entries are erased
// from old partition. The callback should perform NDT update, then sleep long enough for
// previous dispatched packets that depend on old entries to be processed. Note that the FIB
// could not process other commands during this sleep period. In case the callback errors,
// relocating operation will be reverted.
type RelocateCallback func(nRelocated int) error

// Relocate entries under an NDT index from one partition to another.
func (fib *Fib) Relocate(ndtIndex uint64, oldPartition, newPartition uint8,
	cb RelocateCallback) (e error) {
	if int(oldPartition) >= len(fib.parts) {
		return errors.New("bad old partition")
	}
	if int(newPartition) >= len(fib.parts) {
		return errors.New("bad new partition")
	}

	logEntry := log.WithFields(makeLogFields("ndtIndex", ndtIndex,
		"oldPartition", oldPartition, "newPartition", newPartition))
	if oldPartition == newPartition {
		logEntry.Info("Relocate noop")
		return nil
	}

	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		oldPart := fib.parts[oldPartition]
		newPart := fib.parts[newPartition]

		// find old entries
		var oldEntries []*C.FibEntry
		nDyns := 0
		fib.tree.TraverseSubtree(ndtIndex, func(name *ndn.Name, n *fibtree.Node) bool {
			if n.IsEntry || (n.MaxDepth > 0 && name.Len() == fib.cfg.StartDepth) {
				oldEntry := oldPart.Find(name)
				if oldEntry == nil {
					panic(fmt.Errorf("entry %s missing in old partition", name))
				}
				oldEntries = append(oldEntries, oldEntry)
				if oldEntry.dyn != nil {
					nDyns++
				}
			}
			return true
		})
		logEntry = logEntry.WithFields(makeLogFields("nEntries", len(oldEntries), "nDyns", nDyns))

		// allocate new entries
		newDyns := make([]*C.FibEntryDyn, nDyns)
		if e := newPart.dynMp.AllocBulk(newDyns); e != nil {
			return e
		}
		newEntries := make([]*C.FibEntry, len(oldEntries))
		if len(oldEntries) > 0 {
			if ok := bool(C.Fib_AllocBulk(newPart.c, &newEntries[0], C.unsigned(len(newEntries)))); !ok {
				newPart.dynMp.FreeBulk(newDyns)
				return errors.New("new entries allocation error")
			}
		}

		// insert new entries
		nextNewDyn := 0
		for i, oldEntry := range oldEntries {
			newEntry := newEntries[i]
			*newEntry = *oldEntry
			if oldEntry.dyn != nil {
				newEntry.dyn = newDyns[nextNewDyn]
				nextNewDyn++
				C.FibEntryDyn_Copy(newEntry.dyn, oldEntry.dyn)
			}
			if isNew := newPart.Insert(newEntry); !isNew {
				entry := Entry{*oldEntry}
				panic(fmt.Errorf("unexpected entry %s in new partition", entry.GetName()))
			}
		}

		// invoke callback, revert on error
		if cbErr := cb(len(oldEntries)); cbErr != nil {
			// revert: erase new entries
			for _, newEntry := range newEntries {
				newPart.Erase(newEntry)
			}
		}

		// erase old entries
		for _, oldEntry := range oldEntries {
			oldPart.Erase(oldEntry)
		}
		return nil
	})

	if e != nil {
		logEntry.WithError(e).Error("Relocate")
	} else {
		logEntry.Info("Relocate")
	}
	return e
}
