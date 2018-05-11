package fib

/*
#include "fib.h"
*/
import "C"
import (
	"errors"

	"ndn-dpdk/core/urcu"
	"ndn-dpdk/ndn"
)

// Insert a FIB entry.
// If an existing entry has the same name, it will be replaced.
func (fib *Fib) Insert(entry *Entry) (isNew bool, e error) {
	if entry.c.nNexthops == 0 {
		return false, errors.New("cannot insert FIB entry with no nexthop")
	}
	if entry.c.strategy == nil {
		return false, errors.New("cannot insert FIB entry without strategy")
	}

	e = fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		name := entry.GetName()
		nComps := name.Len()
		logEntry := log.WithFields(makeLogFields("name", name, "nexthops", entry.GetNexthops(), "strategy", entry.GetStrategy()))

		newEntryC := C.Fib_Alloc(fib.c)
		if newEntryC == nil {
			logEntry.Error("Insert err=entry-alloc-err")
			return errors.New("FIB entry allocation error")
		}

		if name.Len() > fib.startDepth {
			virtNameV := ndn.JoinNameComponents(name.ListPrefixComps(fib.startDepth))
			oldVirtC := fib.findC(virtNameV)
			if oldVirtC == nil || int(oldVirtC.maxDepth) < nComps-fib.startDepth {
				newVirtC := C.Fib_Alloc(fib.c)
				if newVirtC == nil {
					logEntry.Error("Insert err=virt-alloc-err")
					C.Fib_Free(fib.c, newEntryC)
					return errors.New("FIB virtual entry allocation error")
				}
				if oldVirtC == nil {
					entrySetName(newVirtC, virtNameV, fib.startDepth)
					fib.nVirtuals++
				} else {
					*newVirtC = *oldVirtC
				}
				newVirtC.maxDepth = C.uint8_t(nComps - fib.startDepth)
				logEntry = logEntry.WithFields(makeLogFields("old-virt", addressOf(oldVirtC), "new-virt", addressOf(newVirtC), "max-depth", newVirtC.maxDepth))
				C.Fib_Insert(fib.c, newVirtC)
			} else {
				logEntry = logEntry.WithField("reuse-virt", addressOf(oldVirtC))
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

		logEntry = logEntry.WithField("entry", addressOf(newEntryC))
		if bool(C.Fib_Insert(fib.c, newEntryC)) || isReplacingVirtual {
			if isReplacingVirtual {
				logEntry.Info("Insert replace-virt")
			} else {
				logEntry.Info("Insert new-entry")
			}
			fib.nEntries++
			fib.tree.Insert(name)
			isNew = true
		} else {
			logEntry.Info("Insert replace-entry")
			isNew = false
		}
		return nil
	})
	return isNew, e
}

// Erase a FIB entry by name.
func (fib *Fib) Erase(name *ndn.Name) error {
	return fib.postCommand(func(rs *urcu.ReadSide) error {
		rs.Lock()
		defer rs.Unlock()
		nComps := name.Len()
		logEntry := log.WithField("name", name)

		oldEntryC := fib.findC(name.GetValue())
		if oldEntryC == nil || oldEntryC.nNexthops == 0 {
			logEntry.Error("Erase err=no-entry")
			return errors.New("FIB entry does not exist")
		}
		oldMd, newMd := fib.tree.Erase(name, fib.startDepth)
		logEntry = logEntry.WithFields(makeLogFields("old-max-depth", oldMd, "new-max-depth", newMd))

		var oldVirtC *C.FibEntry
		if nComps > fib.startDepth && oldMd != newMd {
			virtNameV := ndn.JoinNameComponents(name.ListPrefixComps(fib.startDepth))
			oldVirtC = fib.findC(virtNameV)
		} else if nComps == fib.startDepth && newMd != 0 {
			oldVirtC = oldEntryC
			oldEntryC = nil // don't delete, because newVirtC is replacing oldEntryC
		}

		if oldVirtC != nil {
			if newMd != 0 { // need to replace virtual entry
				newVirtC := C.Fib_Alloc(fib.c)
				if newVirtC == nil {
					logEntry.Error("Erase err=virt-alloc-err")
					fib.tree.Insert(name) // revert tree change
					return errors.New("FIB virtual entry allocation error")
				}
				logEntry = logEntry.WithFields(makeLogFields("old-virt", addressOf(oldVirtC), "new-virt", addressOf(newVirtC)))

				*newVirtC = *oldVirtC
				newVirtC.maxDepth = C.uint8_t(newMd)
				C.Fib_Insert(fib.c, newVirtC)

				if (newVirtC.nNexthops == 0 && oldMd == 0 && newMd > 0) || oldEntryC == nil {
					fib.nVirtuals++
				}
			} else if oldVirtC.nNexthops == 0 { // need to erase virtual entry
				logEntry = logEntry.WithField("erase-virt", addressOf(oldVirtC))
				C.Fib_Erase(fib.c, oldVirtC)
				fib.nVirtuals--
			}
		}

		fib.nEntries--
		if oldEntryC != nil {
			logEntry = logEntry.WithField("erase-entry", addressOf(oldEntryC))
			C.Fib_Erase(fib.c, oldEntryC)
		}

		logEntry.Info("Erase")
		return nil
	})
}
