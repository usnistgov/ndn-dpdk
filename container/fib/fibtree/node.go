package fibtree

import (
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type component struct {
	typ   uint16
	value string
}

func (c component) NameComponent() (nc ndn.NameComponent) {
	nc.Type = uint32(c.typ)
	nc.Value = []byte(c.value)
	return
}

type node struct {
	fibdef.EntryBody
	height int

	parent   *node
	comp     component
	children map[component]*node
}

func (n *node) AddChild(comp component, child *node) {
	child.comp = comp
	child.parent = n
	n.children[comp] = child
}

func (n *node) RemoveChild(child *node) {
	delete(n.children, child.comp)
	child.parent = nil
}

func (n *node) UpdateHeight() {
	h := -1
	for _, child := range n.children {
		h = max(h, child.height)
	}
	n.height = h + 1
}

func (n *node) Name() (name ndn.Name) {
	if n.parent == nil {
		return make(ndn.Name, 0, n.height)
	}
	return append(n.parent.Name(), n.comp.NameComponent())
}

func (n *node) IsEntry() bool {
	return len(n.Nexthops) > 0
}

func (n *node) AppendListTo(parentName ndn.Name, list *[]fibdef.Entry) {
	var name ndn.Name
	if n.parent == nil {
		name = ndn.Name{}
	} else {
		name = make(ndn.Name, len(parentName)+1)
		copy(name, parentName)
		name[len(parentName)] = n.comp.NameComponent()
	}

	if n.IsEntry() {
		entry := fibdef.Entry{
			EntryBody: n.EntryBody,
			Name:      name,
		}
		*list = append(*list, entry)
	}

	for _, child := range n.children {
		child.AppendListTo(name, list)
	}
}

func newNode() *node {
	return &node{
		children: map[component]*node{},
	}
}
