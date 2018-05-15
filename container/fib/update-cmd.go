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
		if nComps < fib.ndt.GetPrefixLen() {
			logEntry = logEntry.WithField("partition", "all")
			fibsC = fib.c
		} else {
			_, partition := fib.ndt.Lookup(name)
			logEntry = logEntry.WithField("partition", partition)
			if int(partition) >= len(fib.c) {
				return errors.New("bad partition")
			}
			fibsC = []*C.Fib{fib.c[partition]}
		}

		// allocate and populate new entries
		var newEntriesC []*C.FibEntry
		for _, fibC := range fibsC {
			if newEntryC := C.Fib_Alloc(fibC); newEntryC == nil {
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
				newVirtC := C.Fib_Alloc(fibC)
				if newVirtC == nil {
					C.Fib_Free(fibC, newEntriesC[0])
					return errors.New("allocation error")
				}
				if oldVirtC == nil {
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
			fib.insertNode(name)
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
		if nComps < fib.ndt.GetPrefixLen() {
			logEntry = logEntry.WithField("partition", "all")
			fibsC = fib.c
		} else {
			_, partition := fib.ndt.Lookup(name)
			logEntry = logEntry.WithField("partition", partition)
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
		oldMd, newMd := fib.eraseNode(name, fib.startDepth)

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
					newVirtC := C.Fib_Alloc(fibC)
					if newVirtC == nil {
						fib.insertNode(name) // revert tree change
						return errors.New("allocation error")
					}
					*newVirtC = *oldVirtC
					newVirtC.maxDepth = C.uint8_t(newMd)
					fib.insertC(fibC, newVirtC)
				}
			} else if nComps == fib.startDepth && newMd != 0 {
				// replace oldEntriesC[0] with virtual entry
				newVirtC := C.Fib_Alloc(fibC)
				if newVirtC == nil {
					fib.insertNode(name)
					return errors.New("allocation error")
				}
				entrySetName(newVirtC, name.GetValue(), nComps)
				newVirtC.maxDepth = C.uint8_t(newMd)
				fib.insertC(fibC, newVirtC)
				oldEntriesC = nil // don't delete oldEntriesC[0]
			}
		}

		for i, oldEntryC := range oldEntriesC {
			fib.eraseC(fibsC[i], oldEntryC)
		}
		return nil
	})

	if e != nil {
		logEntry.WithError(e).Error("Erase")
	} else {
		logEntry.Info("Erase")
	}
	return e
}
