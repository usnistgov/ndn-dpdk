package fibtree

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// Node represents a node in Tree.
type Node struct {
	// IsEntry is set to true if there is a FIB entry at this node.
	IsEntry bool

	// MaxDepth indicates the height of a subtree rooted at this node.
	MaxDepth int

	// children maps from NameComponent TLV-VALUE to child node.
	children map[string]*Node
}

func newNode() *Node {
	return &Node{children: make(map[string]*Node)}
}

func (n *Node) updateMaxDepth() {
	n.MaxDepth = 0
	for _, child := range n.children {
		n.MaxDepth = math.MaxInt(n.MaxDepth, 1+child.MaxDepth)
	}
}

// Visit this node and its descendants in preorder traversal.
func (n *Node) traverse(nameV string, cb TraverseCallback) {
	var name ndn.Name
	name.UnmarshalBinary([]byte(nameV))
	visitChildren := cb(name, n)
	if !visitChildren {
		return
	}
	for comp, child := range n.children {
		child.traverse(nameV+comp, cb)
	}
}
