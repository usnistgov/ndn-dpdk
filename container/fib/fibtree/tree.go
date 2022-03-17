// Package fibtree organizes logical FIB entries in a name hierarchy.
package fibtree

import (
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type update struct {
	real   *fibdef.RealUpdate
	virt   *fibdef.VirtUpdate
	revert func()
}

var _ fibdef.Update = update{}

func (u update) Real() *fibdef.RealUpdate {
	return u.real
}

func (u update) Virt() *fibdef.VirtUpdate {
	return u.virt
}

func (u update) Commit() {
}

func (u update) Revert() {
	if u.revert != nil {
		u.revert()
	}
}

// Tree represents a tree of name hierarchy.
type Tree struct {
	root       *node
	startDepth int

	nNodes   int
	nEntries int
}

// CountNodes returns number of nodes.
func (t *Tree) CountNodes() int {
	return t.nNodes
}

// CountEntries returns number of entries.
func (t *Tree) CountEntries() int {
	return t.nEntries
}

func (t *Tree) seek(name ndn.Name, canInsert bool) (n *node, isNew bool) {
	n = t.root
	for _, comp := range name {
		compS := componentToString(comp)
		child := n.children[compS]
		if child == nil {
			if !canInsert {
				return nil, false
			}
			child = newNode()
			n.addChild(compS, child)
			t.nNodes++
			isNew = true
		}
		n = child
	}
	return
}

// List lists entries.
func (t *Tree) List() (list []fibdef.Entry) {
	t.root.appendListTo("", &list)
	return
}

// Find retrieves an entry by exact match.
func (t *Tree) Find(name ndn.Name) *fibdef.Entry {
	n, _ := t.seek(name, false)
	if n == nil || !n.isEntry() {
		return nil
	}
	return &fibdef.Entry{
		Name:      name,
		EntryBody: n.EntryBody,
	}
}

// Insert inserts or replaces an entry.
func (t *Tree) Insert(entry fibdef.Entry) fibdef.Update {
	if e := entry.Validate(); e != nil {
		panic(e)
	}

	u := update{}

	n, isNewNode := t.seek(entry.Name, true)
	if fibdef.EntryBodyEquals(n.EntryBody, entry.EntryBody) {
		// if n.EntryBody is the same, it cannot be a new node
		return u
	}

	u.real = &fibdef.RealUpdate{
		Name:      entry.Name,
		Action:    fibdef.ActReplace,
		EntryBody: entry.EntryBody,
	}
	if n.isEntry() {
		oldEntry := fibdef.Entry{
			Name:      entry.Name,
			EntryBody: n.EntryBody,
		}
		u.revert = func() { t.Insert(oldEntry) }
	} else {
		u.revert = func() { t.Erase(entry.Name) }

		u.real.Action = fibdef.ActInsert
		t.nEntries++
	}
	n.EntryBody = entry.EntryBody

	if len(entry.Name) == t.startDepth && n.height > 0 {
		u.real.WithVirt = &fibdef.VirtUpdate{
			Name:    entry.Name,
			Action:  fibdef.ActReplace,
			HasReal: true,
			Height:  n.height,
		}

		// if n.height is non-zero, it must have children and is not a new node
		return u
	}

	if isNewNode {
		// update height on ancestors; height of the new node remains zero
		for depth, p := len(entry.Name), n.parent; p != nil; p = p.parent {
			depth--
			oldHeight := p.height
			p.updateHeight()
			if depth == t.startDepth && oldHeight != p.height {
				u.virt = &fibdef.VirtUpdate{
					Name:    p.name(),
					Action:  fibdef.ActReplace,
					HasReal: p.isEntry(),
					Height:  p.height,
				}
				if oldHeight == 0 {
					u.virt.Action = fibdef.ActInsert
				}
			}
		}
	}
	return u
}

// Erase deletes an entry.
func (t *Tree) Erase(name ndn.Name) fibdef.Update {
	var u update

	n, _ := t.seek(name, false)
	if n == nil || !n.isEntry() {
		return u
	}
	oldEntry := fibdef.Entry{
		Name:      name,
		EntryBody: n.EntryBody,
	}
	u.revert = func() { t.Insert(oldEntry) }
	t.nEntries--
	n.EntryBody = fibdef.EntryBody{}

	u.real = &fibdef.RealUpdate{
		Name:   name,
		Action: fibdef.ActErase,
	}
	if len(name) == t.startDepth && n.height > 0 {
		u.real.WithVirt = &fibdef.VirtUpdate{
			Name:    name,
			Action:  fibdef.ActReplace,
			HasReal: false,
			Height:  n.height,
		}

		// if n.height is non-zero, it must have children and is not deletable
		return u
	}

	for depth, c, p := len(name), n, n.parent; p != nil && !c.isEntry() && len(c.children) == 0; c, p = p, p.parent {
		depth--
		p.removeChild(c)
		t.nNodes--

		oldHeight := p.height
		p.updateHeight()
		if depth == t.startDepth && oldHeight != p.height {
			u.virt = &fibdef.VirtUpdate{
				Name:    p.name(),
				Action:  fibdef.ActReplace,
				HasReal: p.isEntry(),
				Height:  p.height,
			}
			if p.height == 0 {
				u.virt.Action = fibdef.ActErase
			}
		}
	}
	return u
}

// New creates a Tree.
func New(startDepth int) *Tree {
	return &Tree{
		root:       newNode(),
		startDepth: startDepth,
		nNodes:     1,
	}
}
