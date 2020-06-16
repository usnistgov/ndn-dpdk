package fibtree

import (
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type Node struct {
	IsEntry  bool
	MaxDepth int
	children map[string]*Node // string(NameComponent) => child
}

func newNode() *Node {
	return &Node{children: make(map[string]*Node)}
}

func (n *Node) updateMaxDepth() {
	n.MaxDepth = 0
	for _, child := range n.children {
		depth := 1 + child.MaxDepth
		if depth > n.MaxDepth {
			n.MaxDepth = depth
		}
	}
}

// Visit this node and its descendants in preorder traversal.
func (n *Node) traverse(nameV string, cb TraverseCallback) {
	name, _ := ndni.NewName(ndni.TlvBytes(nameV))
	visitChildren := cb(name, n)
	if !visitChildren {
		return
	}
	for comp, child := range n.children {
		child.traverse(nameV+comp, cb)
	}
}
