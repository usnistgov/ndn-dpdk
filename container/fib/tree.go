package fib

import (
	"ndn-dpdk/ndn"
)

type node struct {
	IsEntry  bool
	Comp     ndn.NameComponent
	MaxDepth int
	Children map[*node]struct{}
}

func (n *node) FindChild(comp ndn.NameComponent) *node {
	for child := range n.Children {
		if comp.Equal(child.Comp) {
			return child
		}
	}
	return nil
}

func (n *node) UpdateMaxDepth() {
	n.MaxDepth = 0
	for child := range n.Children {
		depth := 1 + child.MaxDepth
		if depth > n.MaxDepth {
			n.MaxDepth = depth
		}
	}
}

func (n *node) ListTo(names *[]*ndn.Name, prefix ndn.TlvBytes) {
	nameV := append(append(ndn.TlvBytes(nil), prefix...), n.Comp...)
	if n.IsEntry {
		name, _ := ndn.NewName(nameV)
		*names = append(*names, name)
	}
	for child := range n.Children {
		child.ListTo(names, nameV)
	}
}

type tree node

func (t *tree) seek(comps []ndn.NameComponent, wantInsert bool) (nodes []*node) {
	nodes = make([]*node, len(comps)+1)
	nodes[0] = (*node)(t)

	for i, comp := range comps {
		parent := nodes[i]
		child := parent.FindChild(comp)
		if child == nil {
			if !wantInsert {
				return nodes[:i+1]
			}
			if parent.Children == nil {
				parent.Children = make(map[*node]struct{})
			}
			child = new(node)
			child.Comp = comp
			parent.Children[child] = struct{}{}
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
			delete(nodes[i-1].Children, node)
		}
	}
	return
}

func (t *tree) List() (names []*ndn.Name) {
	names = make([]*ndn.Name, 0)
	(*node)(t).ListTo(&names, nil)
	return names
}
