package fibreplica

import "github.com/usnistgov/ndn-dpdk/container/fib/fibdef"

// UpdateCommand represents a prepared update command.
type UpdateCommand struct {
	real       realUpdate
	virt       virtUpdate
	allocated  []*Entry
	allocSplit int
}

func (u *UpdateCommand) clear() {
	u.real.RealUpdate = nil
	u.virt.VirtUpdate = nil
	u.allocated = nil
}

// PrepareUpdate prepares an update.
func (t *Table) PrepareUpdate(tu fibdef.Update) (*UpdateCommand, error) {
	u := &UpdateCommand{}
	u.real.RealUpdate = tu.Real()
	u.virt.VirtUpdate = tu.Virt()

	u.allocSplit = u.real.prepare(t)
	u.allocated = make([]*Entry, u.allocSplit+u.virt.prepare(t))
	if e := t.allocBulk(u.allocated); e != nil {
		return nil, e
	}

	return u, nil
}

// ExecuteUpdate applies an update.
func (t *Table) ExecuteUpdate(u *UpdateCommand) {
	u.real.execute(t, u.allocated[:u.allocSplit])
	u.virt.execute(t, u.allocated[u.allocSplit:])
	u.clear()
}

// DiscardUpdate releases resources in an unexecuted update.
func (t *Table) DiscardUpdate(u *UpdateCommand) {
	if u.allocated != nil {
		t.mp.Free(u.allocated)
	}
	u.clear()
}

type realUpdate struct {
	*fibdef.RealUpdate
	oldReal, oldVirt *Entry
	newReal, newVirt *Entry
}

func (u *realUpdate) prepare(t *Table) (nAlloc int) {
	if u.RealUpdate == nil {
		return 0
	}
	if u.WithVirt != nil && u.WithVirt.Action != fibdef.ActReplace {
		panic("unexpected u.WithVirt.Action")
	}

	switch u.Action {
	case fibdef.ActInsert:
		nAlloc++
		if u.WithVirt != nil {
			nAlloc++
			u.oldVirt = t.Get(u.Name)
		}
	case fibdef.ActReplace:
		nAlloc++
		if u.WithVirt != nil {
			nAlloc++
			u.oldVirt = t.Get(u.Name)
			u.oldReal = u.oldVirt.Real()
		} else {
			u.oldReal = t.Get(u.Name)
		}
	case fibdef.ActErase:
		if u.WithVirt != nil {
			nAlloc++
			u.oldVirt = t.Get(u.Name)
			u.oldReal = u.oldVirt.Real()
		} else {
			u.oldReal = t.Get(u.Name)
		}
	}
	return nAlloc
}

func (u *realUpdate) execute(t *Table, allocated []*Entry) {
	if u.RealUpdate == nil {
		return
	}

	switch u.Action {
	case fibdef.ActInsert, fibdef.ActReplace:
		u.newReal = allocated[0]
		u.newReal.assignReal(u.RealUpdate)
		if u.WithVirt != nil {
			u.newVirt = allocated[1]
			u.newVirt.assignVirt(u.WithVirt, u.newReal)
			t.write(u.newVirt)
		} else {
			t.write(u.newReal)
		}
	case fibdef.ActErase:
		if u.WithVirt != nil {
			u.newVirt = allocated[0]
			u.newVirt.assignVirt(u.WithVirt, nil)
			t.write(u.newVirt)
		} else if u.oldVirt != nil {
			t.erase(u.oldVirt)
		} else {
			t.erase(u.oldReal)
		}
	}

	if u.oldReal != nil {
		t.deferredFree(u.oldReal)
	}
	if u.oldVirt != nil {
		t.deferredFree(u.oldVirt)
	}
}

type virtUpdate struct {
	*fibdef.VirtUpdate
	oldVirt, oldReal *Entry
	newVirt          *Entry
}

func (u *virtUpdate) prepare(t *Table) (nAlloc int) {
	if u.VirtUpdate == nil {
		return 0
	}

	switch u.Action {
	case fibdef.ActInsert:
		nAlloc++
		if u.HasReal {
			u.oldReal = t.Get(u.Name)
		}
	case fibdef.ActReplace:
		nAlloc++
		u.oldVirt = t.Get(u.Name)
		if u.HasReal {
			u.oldReal = u.oldVirt.Real()
		}
	case fibdef.ActErase:
		u.oldVirt = t.Get(u.Name)
		if u.HasReal {
			u.oldReal = u.oldVirt.Real()
		}
	}
	return nAlloc
}

func (u *virtUpdate) execute(t *Table, allocated []*Entry) {
	if u.VirtUpdate == nil {
		return
	}

	switch u.Action {
	case fibdef.ActInsert, fibdef.ActReplace:
		u.newVirt = allocated[0]
		u.newVirt.assignVirt(u.VirtUpdate, u.oldReal)
		t.write(u.newVirt)
	case fibdef.ActErase:
		if u.oldReal == nil {
			t.erase(u.oldVirt)
		} else {
			t.write(u.oldReal)
		}
	}

	if u.oldVirt != nil {
		t.deferredFree(u.oldVirt)
	}
}
