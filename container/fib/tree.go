package fib

import (
	"ndn-dpdk/ndn"
)

type treeWalkCallback func(name string, isEntry bool)

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

func (n *node) Walk(name string, cb treeWalkCallback) {
	cb(name, n.IsEntry)
	for comp, child := range n.Children {
		child.Walk(name+comp, cb)
	}
}

func (fib *Fib) seekNode(name *ndn.Name, wantInsert bool) (nodes []*node) {
	nodes = make([]*node, name.Len()+1)
	nodes[0] = &fib.treeRoot

	for i := 0; i < name.Len(); i++ {
		parent := nodes[i]
		if parent.Children == nil {
			if wantInsert {
				parent.Children = make(map[string]*node)
			} else {
				return nodes[:i+1]
			}
		}
		comp := string(name.GetComp(i))
		child := parent.Children[comp]
		if child == nil {
			if !wantInsert {
				return nodes[:i+1]
			}
			child = new(node)
			parent.Children[comp] = child
		}
		nodes[i+1] = child
	}
	return nodes
}

func (fib *Fib) insertNode(name *ndn.Name) {
	nodes := fib.seekNode(name, true)
	nodes[name.Len()].IsEntry = true

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		node.UpdateMaxDepth()
	}
}

func (fib *Fib) eraseNode(name *ndn.Name, startDepth int) (oldMd int, newMd int) {
	nodes := fib.seekNode(name, false)
	nodes[name.Len()].IsEntry = false // will panic if node does not exist

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
			delete(nodes[i-1].Children, string(name.GetComp(i-1)))
		}
	}
	return
}
