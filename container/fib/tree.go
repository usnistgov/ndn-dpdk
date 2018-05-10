package fib

import (
	"ndn-dpdk/ndn"
)

type node struct {
	IsEntry  bool
	MaxDepth int
	Children map[string]*node // key is NameComponent cast as string
}

func (n *node) UpdateMaxDepth() {
	n.MaxDepth = 0
	for _, child := range n.Children {
		depth := 1 + child.MaxDepth
		if depth > n.MaxDepth {
			n.MaxDepth = depth
		}
	}
}

func (n *node) ListTo(names *[]*ndn.Name, prefix string) {
	if n.IsEntry {
		name, _ := ndn.NewName(ndn.TlvBytes(prefix))
		*names = append(*names, name)
	}
	for comp, child := range n.Children {
		child.ListTo(names, prefix+comp)
	}
}

type tree node

func (t *tree) seek(comps []ndn.NameComponent, wantInsert bool) (nodes []*node) {
	nodes = make([]*node, len(comps)+1)
	nodes[0] = (*node)(t)

	for i, comp := range comps {
		parent := nodes[i]
		if parent.Children == nil {
			if wantInsert {
				parent.Children = make(map[string]*node)
			} else {
				return nodes[:i+1]
			}
		}
		compStr := string(comp)
		child := parent.Children[compStr]
		if child == nil {
			if !wantInsert {
				return nodes[:i+1]
			}
			child = new(node)
			parent.Children[compStr] = child
		}
		nodes[i+1] = child
	}
	return nodes
}

func (t *tree) Insert(comps []ndn.NameComponent) {
	nodes := t.seek(comps, true)
	nodes[len(comps)].IsEntry = true

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		node.UpdateMaxDepth()
	}
}

func (t *tree) Erase(comps []ndn.NameComponent, startDepth int) (oldMd int, newMd int) {
	nodes := t.seek(comps, false)
	nodes[len(comps)].IsEntry = false // will panic if node does not exist

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		if i == startDepth {
			oldMd = node.MaxDepth
		}
		node.UpdateMaxDepth()
		if i == startDepth {
			newMd = node.MaxDepth
		}
		if i > 0 && !node.IsEntry && len(node.Children) == 0 {
			delete(nodes[i-1].Children, string(comps[i-1]))
		}
	}
	return
}

func (t *tree) List() (names []*ndn.Name) {
	names = make([]*ndn.Name, 0)
	(*node)(t).ListTo(&names, "")
	return names
}
