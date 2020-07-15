package fib

/*
#include "../../csrc/fib/fib.h"
*/
import "C"
import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibtree"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type updateAct int

const (
	updateActInsert updateAct = iota
	updateActInsertNoDiscard
	updateActErase
)

type updateItem struct {
	act      updateAct
	part     *partition
	entry    *Entry
	freeVirt C.Fib_FreeOld
	freeReal C.Fib_FreeOld
}

type updateBatch []updateItem

func (batch updateBatch) Apply() {
	for _, item := range batch {
		switch item.act {
		case updateActInsert, updateActInsertNoDiscard:
			item.part.Insert(item.entry, item.freeVirt, item.freeReal)
		case updateActErase:
			item.part.Erase(item.entry, item.freeVirt, item.freeReal)
		}
	}
}

func (batch updateBatch) Discard(part *partition) error {
	for _, item := range batch {
		if item.act == updateActInsert {
			C.Fib_Free(item.part.c, item.entry.ptr())
		}
	}
	return fmt.Errorf("allocation error in partition %d", part.index)
}

func (fib *Fib) getVirtName(name ndn.Name) ndn.Name {
	if len(name) < fib.cfg.StartDepth {
		return nil
	}
	return name[:fib.cfg.StartDepth]
}

// Insert inserts a FIB entry, or replaces an existing entry with the same name.
func (fib *Fib) Insert(entry *Entry) (isNew bool, e error) {
	if len(entry.Nexthops()) == 0 {
		return false, errors.New("cannot insert FIB entry with no nexthop")
	}
	if entry.Strategy() == nil {
		return false, errors.New("cannot insert FIB entry with no strategy")
	}
	name := entry.Name()
	virtName := fib.getVirtName(name)
	logEntry := log.WithFields(makeLogFields("name", name, "nexthops", entry.Nexthops(), "strategy", entry.Strategy().GetId()))

	e, _ = eal.CallMain(func() error {
		eal.MainReadSide.Lock()
		defer eal.MainReadSide.Unlock()

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
			newEntry := part.Alloc(name)
			if newEntry == nil {
				return batch.Discard(part)
			}
			newEntry.copyFrom(entry)
			batch = append(batch, updateItem{updateActInsert, part, newEntry, C.Fib_FreeOld_MustNotExist, C.Fib_FreeOld_YesIfExists})

			switch {
			case len(name) < fib.cfg.StartDepth:
				// virtual entry not involved

			case len(name) == fib.cfg.StartDepth && newMd == 0:
				// no virtual entry necessary

			case len(name) == fib.cfg.StartDepth && newMd > 0:
				// insert virtual entry before real entry
				newVirt := part.Alloc(virtName)
				if newVirt == nil {
					return batch.Discard(part)
				}
				newVirt.setMaxDepthReal(newMd, newEntry)
				batch[len(batch)-1] = updateItem{updateActInsert, part, newVirt, C.Fib_FreeOld_YesIfExists, C.Fib_FreeOld_YesIfExists}

			case len(name) > fib.cfg.StartDepth && oldMd == newMd:
				// no virtual entry update necessary

			case len(name) > fib.cfg.StartDepth && oldMd != newMd && !virtIsEntry:
				// insert or replace virtual entry; no real entry at virtName
				newVirt := part.Alloc(virtName)
				if newVirt == nil {
					return batch.Discard(part)
				}
				newVirt.setMaxDepthReal(newMd, nil)
				batch = append(batch, updateItem{updateActInsert, part, newVirt, C.Fib_FreeOld_YesIfExists, C.Fib_FreeOld_MustNotExist})

			case len(name) > fib.cfg.StartDepth && oldMd != newMd && virtIsEntry:
				// insert or replace virtual entry before existing real entry at virtName
				oldReal := part.Get(virtName).getReal()
				if oldReal == nil {
					panic(fmt.Errorf("real entry %s missing in partition %d", virtName, part.index))
				}
				newVirt := part.Alloc(virtName)
				if newVirt == nil {
					return batch.Discard(part)
				}
				newVirt.setMaxDepthReal(newMd, oldReal)
				batch = append(batch, updateItem{updateActInsert, part, newVirt, C.Fib_FreeOld_YesIfExists, C.Fib_FreeOld_No})

			default:
				panic("unexpected case")
			}
		}

		// perform batch updates
		batch.Apply()
		success = true
		return nil
	}).(error)

	if e != nil {
		logEntry.WithError(e).Error("Insert")
	} else {
		logEntry.Info("Insert")
	}
	return isNew, e
}

// Erase a FIB entry by name.
func (fib *Fib) Erase(name ndn.Name) (e error) {
	virtName := fib.getVirtName(name)
	logEntry := log.WithField("name", name)

	e, _ = eal.CallMain(func() error {
		eal.MainReadSide.Lock()
		defer eal.MainReadSide.Unlock()

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
			oldEntry := part.Get(name)
			if oldEntry == nil {
				panic(fmt.Errorf("entry %s missing in partition %d", name, part.index))
			}
			batch = append(batch, updateItem{updateActErase, part, oldEntry, C.Fib_FreeOld_MustNotExist, C.Fib_FreeOld_Yes})

			switch {
			case len(name) < fib.cfg.StartDepth:
				// virtual entry not involved

			case len(name) == fib.cfg.StartDepth && newMd == 0:
				// erase real entry; erase virtual entry if exists
				batch[len(batch)-1] = updateItem{updateActErase, part, oldEntry, C.Fib_FreeOld_YesIfExists, C.Fib_FreeOld_Yes}

			case len(name) == fib.cfg.StartDepth && newMd > 0:
				// erase real entry; keep virtual entry by inserting another virtual entry
				newVirt := part.Alloc(virtName)
				if newVirt == nil {
					return batch.Discard(part)
				}
				newVirt.setMaxDepthReal(newMd, nil)
				batch[len(batch)-1] = updateItem{updateActInsert, part, newVirt, C.Fib_FreeOld_Yes, C.Fib_FreeOld_Yes}

			case len(name) > fib.cfg.StartDepth && oldMd == newMd:
				// no virtual entry update necessary

			case len(name) > fib.cfg.StartDepth && oldMd != newMd && newMd == 0 && !virtIsEntry:
				// erase virtual entry; no real entry at virtName
				oldVirt := part.Get(virtName)
				if !oldVirt.isVirt() {
					panic(fmt.Errorf("virtual entry %s missing in partition %d", name, part.index))
				}
				batch = append(batch, updateItem{updateActErase, part, oldVirt, C.Fib_FreeOld_Yes, C.Fib_FreeOld_MustNotExist})

			case len(name) > fib.cfg.StartDepth && oldMd != newMd && newMd == 0 && virtIsEntry:
				// erase virtual entry; keep real entry at virtName
				oldReal := part.Get(virtName).getReal()
				if oldReal == nil {
					panic(fmt.Errorf("real entry %s missing in partition %d", virtName, part.index))
				}
				batch = append(batch, updateItem{updateActInsertNoDiscard, part, oldReal, C.Fib_FreeOld_Yes, C.Fib_FreeOld_No})

			case len(name) > fib.cfg.StartDepth && oldMd != newMd && newMd > 0 && !virtIsEntry:
				// replace virtual entry; no real entry at virtName
				newVirt := part.Alloc(virtName)
				if newVirt == nil {
					return batch.Discard(part)
				}
				newVirt.setMaxDepthReal(newMd, nil)
				batch = append(batch, updateItem{updateActInsert, part, newVirt, C.Fib_FreeOld_Yes, C.Fib_FreeOld_MustNotExist})

			case len(name) > fib.cfg.StartDepth && oldMd != newMd && newMd > 0 && virtIsEntry:
				// replace virtual entry; keep real entry at virtName
				oldReal := part.Get(virtName).getReal()
				if oldReal == nil {
					panic(fmt.Errorf("real entry %s missing in partition %d", virtName, part.index))
				}
				newVirt := part.Alloc(virtName)
				if newVirt == nil {
					return batch.Discard(part)
				}
				newVirt.setMaxDepthReal(newMd, oldReal)
				batch = append(batch, updateItem{updateActInsert, part, newVirt, C.Fib_FreeOld_Yes, C.Fib_FreeOld_No})

			default:
				panic("unexpected case")
			}
		}

		// perform batch updates
		batch.Apply()
		success = true
		return nil
	}).(error)

	if e != nil {
		logEntry.WithError(e).Error("Erase")
	} else {
		logEntry.Info("Erase")
	}
	return e
}

// RelocateCallback is a callback used during relocate operation.
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

	e, _ = eal.CallMain(func() error {
		eal.MainReadSide.Lock()
		defer eal.MainReadSide.Unlock()
		oldPart := fib.parts[oldPartition]
		newPart := fib.parts[newPartition]

		// prepare batches
		hasAllocErr := false
		var newBatch updateBatch
		var oldBatch updateBatch
		var revertBatch updateBatch
		fib.tree.TraverseSubtree(ndtIndex, func(name ndn.Name, n *fibtree.Node) bool {
			if hasAllocErr {
				return false
			}

			var oldReal, newReal, oldVirt, newVirt *Entry

			if n.IsEntry {
				oldReal = oldPart.Get(name).getReal()
				if oldReal == nil || oldReal.isVirt() {
					panic(fmt.Errorf("real entry %s missing in old partition", name))
				}
				newReal = newPart.Alloc(name)
				if newReal == nil {
					hasAllocErr = true
					return false
				}
				newReal.copyFrom(oldReal)
			}

			if len(name) == fib.cfg.StartDepth && n.MaxDepth > 0 {
				oldVirt = oldPart.Get(name)
				if !oldVirt.isVirt() {
					panic(fmt.Errorf("virtual entry %s missing in old partition", name))
				}
				newVirt = newPart.Alloc(name)
				if newVirt == nil {
					hasAllocErr = true
					return false
				}
				newVirt.setMaxDepthReal(n.MaxDepth, newReal)
			}

			if newVirt != nil {
				oldBatch = append(oldBatch, updateItem{updateActErase, oldPart, oldVirt, C.Fib_FreeOld_Yes, C.Fib_FreeOld_YesIfExists})
				newBatch = append(newBatch, updateItem{updateActInsert, newPart, newVirt, C.Fib_FreeOld_MustNotExist, C.Fib_FreeOld_MustNotExist})
				revertBatch = append(revertBatch, updateItem{updateActErase, newPart, newVirt, C.Fib_FreeOld_YesIfExists, C.Fib_FreeOld_YesIfExists})
			} else if newReal != nil {
				oldBatch = append(oldBatch, updateItem{updateActErase, oldPart, oldReal, C.Fib_FreeOld_MustNotExist, C.Fib_FreeOld_Yes})
				newBatch = append(newBatch, updateItem{updateActInsert, newPart, newReal, C.Fib_FreeOld_MustNotExist, C.Fib_FreeOld_MustNotExist})
				revertBatch = append(revertBatch, updateItem{updateActErase, newPart, newReal, C.Fib_FreeOld_YesIfExists, C.Fib_FreeOld_YesIfExists})
			}
			return true
		})
		if hasAllocErr {
			return newBatch.Discard(newPart)
		}

		nRelocated := len(newBatch)
		logEntry = logEntry.WithFields(makeLogFields("nRelocated", nRelocated))

		// insert new entries
		newBatch.Apply()

		// invoke callback, revert on error
		if cbErr := cb(nRelocated); cbErr != nil {
			// revert: erase new entries
			revertBatch.Apply()
		}

		// erase old entries
		oldBatch.Apply()
		return nil
	}).(error)

	if e != nil {
		logEntry.WithError(e).Error("Relocate")
	} else {
		logEntry.Info("Relocate")
	}
	return e
}
