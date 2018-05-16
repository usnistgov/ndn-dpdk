package fib

import (
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/ndn"
)

type nodeName struct {
	NameV  string
	NComps int
}

func (nn *nodeName) GetName() *ndn.Name {
	n, _ := ndn.NewName(ndn.TlvBytes(nn.NameV))
	return n
}

type node struct {
	IsEntry  bool
	MaxDepth int
	Children map[string]*node // key is NameComponent cast as string
}

func newNode() *node {
	return &node{Children: make(map[string]*node)}
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

type treeWalkCallback func(nn nodeName, node *node)

func (n *node) Walk(nn nodeName, cb treeWalkCallback) {
	cb(nn, n)
	for comp, child := range n.Children {
		child.Walk(nodeName{nn.NameV + comp, nn.NComps + 1}, cb)
	}
}

type subtreeIndex []map[*node]ndn.TlvBytes

func newSubtreeIndex(ndt *ndt.Ndt) subtreeIndex {
	return make(subtreeIndex, ndt.CountElements())
}

func (sti subtreeIndex) Insert(ndtIndex uint64, nameV ndn.TlvBytes, n *node) {
	if sti[ndtIndex] == nil {
		sti[ndtIndex] = make(map[*node]ndn.TlvBytes)
	} else if _, ok := sti[ndtIndex][n]; ok {
		panic("node already in subtreeIndex")
	}
	sti[ndtIndex][n] = nameV
}

func (sti subtreeIndex) Erase(ndtIndex uint64, n *node) {
	if sti[ndtIndex] != nil {
		if _, ok := sti[ndtIndex][n]; ok {
			delete(sti[ndtIndex], n)
			return
		}
	}
	panic("node not in subtreeIndex")
}

func (fib *Fib) CountNodes() int {
	return fib.nNodes
}

func (fib *Fib) updateNEntries(name *ndn.Name, diff int) {
	if name.Len() < fib.ndt.GetPrefixLen() {
		fib.nShortEntries += diff
	} else {
		fib.nLongEntries += diff
	}
}

func (fib *Fib) insertNode(name *ndn.Name, ndtIndex uint64) {
	nodes := []*node{fib.treeRoot}
	for i := 0; i < name.Len(); i++ {
		parent := nodes[i]
		comp := string(name.GetComp(i))
		child := parent.Children[comp]
		if child == nil {
			child = newNode()
			fib.nNodes++
			parent.Children[comp] = child
			if i+1 == fib.ndt.GetPrefixLen() {
				prefixV := ndn.JoinNameComponents(name.ListPrefixComps(i + 1))
				fib.sti.Insert(ndtIndex, prefixV, child)
			}
		}
		nodes = append(nodes, child)
	}

	if nodes[name.Len()].IsEntry {
		panic("node is entry")
	}
	nodes[name.Len()].IsEntry = true
	fib.updateNEntries(name, 1)

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		node.UpdateMaxDepth()
	}
}

func (fib *Fib) eraseNode(name *ndn.Name, ndtIndex uint64) (oldMd int, newMd int) {
	nodes := []*node{fib.treeRoot}
	for i := 0; i < name.Len(); i++ {
		parent := nodes[i]
		child := parent.Children[string(name.GetComp(i))]
		if child == nil {
			panic("node not found")
		}
		nodes = append(nodes, child)
	}

	if !nodes[name.Len()].IsEntry {
		panic("node is not entry")
	}
	nodes[name.Len()].IsEntry = false
	fib.updateNEntries(name, -1)

	for i := len(nodes) - 1; i >= 0; i-- {
		node := nodes[i]
		if i == fib.startDepth {
			oldMd = node.MaxDepth
		}
		node.UpdateMaxDepth()
		if i == fib.startDepth {
			newMd = node.MaxDepth
		}
		if i > 0 && !node.IsEntry && len(node.Children) == 0 {
			if i == fib.ndt.GetPrefixLen() {
				fib.sti.Erase(ndtIndex, node)
			}
			delete(nodes[i-1].Children, string(name.GetComp(i-1)))
			fib.nNodes--
		}
	}
	return
}
