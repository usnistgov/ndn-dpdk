package fibtree

import (
	"bytes"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/zyedidia/generic"
)

func componentToString(comp ndn.NameComponent) string {
	compV, _ := tlv.EncodeFrom(comp)
	return string(compV)
}

type node struct {
	fibdef.EntryBody
	height int

	parent   *node
	comp     string
	children map[string]*node
}

func (n *node) addChild(comp string, child *node) {
	child.comp = comp
	child.parent = n
	n.children[comp] = child
}

func (n *node) removeChild(child *node) {
	delete(n.children, child.comp)
	child.parent = nil
}

func (n *node) updateHeight() {
	h := -1
	for _, child := range n.children {
		h = generic.Max(h, child.height)
	}
	n.height = h + 1
}

func (n *node) name() (name ndn.Name) {
	var buffer bytes.Buffer
	n.appendNameValue(&buffer)
	name.UnmarshalBinary(buffer.Bytes())
	return name
}

func (n *node) appendNameValue(buffer *bytes.Buffer) {
	if n.parent != nil {
		n.parent.appendNameValue(buffer)
		buffer.WriteString(n.comp) // root node has no component
	}
}

func (n *node) isEntry() bool {
	return len(n.Nexthops) > 0
}

func (n *node) appendListTo(parentNameV string, list *[]fibdef.Entry) {
	nameV := parentNameV + n.comp
	if n.isEntry() {
		entry := fibdef.Entry{
			EntryBody: n.EntryBody,
		}
		entry.Name.UnmarshalBinary([]byte(nameV))
		*list = append(*list, entry)
	}

	for _, child := range n.children {
		child.appendListTo(nameV, list)
	}
}

func newNode() *node {
	return &node{
		children: map[string]*node{},
	}
}
