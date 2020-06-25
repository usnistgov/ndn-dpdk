package fibtree

import (
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// GetNdtIndexCallback is a callback function that returns NDT index for the specified name.
type GetNdtIndexCallback func(name ndn.Name) uint64

// TraverseCallback is a visitor during tree traversal.
// Returns whether to visit descendants of current node.
type TraverseCallback func(name ndn.Name, n *Node) bool

// Tree represents a name hierarchy of FIB.
// It contains all the FIB entry names, but does not contain contents (nexthops, etc) of the FIB entry.
type Tree struct {
	startDepth   int
	ndtPrefixLen int
	getNdtIndex  GetNdtIndexCallback

	nEntries int
	nNodes   int
	root     *Node
	subtrees []map[*Node]string // ndtIndex => list of <Node,string(nameV)> tuples, where name.Len()==ndtPrefixLen
}

// New creates a Tree.
func New(startDepth, ndtPrefixLen, nNdtElements int, getNdtIndex GetNdtIndexCallback) (t *Tree) {
	t = new(Tree)
	t.startDepth = startDepth
	t.ndtPrefixLen = ndtPrefixLen
	t.getNdtIndex = getNdtIndex

	t.root = newNode()
	t.nNodes = 1

	t.subtrees = make([]map[*Node]string, nNdtElements)
	for i := range t.subtrees {
		t.subtrees[i] = make(map[*Node]string)
	}
	return t
}

// CountEntries returns number of FIB entries.
func (t *Tree) CountEntries() int {
	return t.nEntries
}

// CountNodes returns number of tree nodes.
func (t *Tree) CountNodes() int {
	return t.nNodes
}

// Traverse traverses the tree starting from the root node.
func (t *Tree) Traverse(cb TraverseCallback) {
	t.root.traverse("", cb)
}

// TraverseSubtree visits subtrees of a specified ndtIndex.
func (t *Tree) TraverseSubtree(ndtIndex uint64, cb TraverseCallback) {
	for n, nameV := range t.subtrees[ndtIndex] {
		n.traverse(nameV, cb)
	}
}

// Insert inserts an entry at the specified name.
// Returns:
//   ok: true if inserted, false if entry already exists
//   oldMd: old MaxDepth at name.GetPrefix(startDepth)
//   newMd: new MaxDepth at name.GetPrefix(startDepth)
//   virtIsEntry: whether node at name.GetPrefix(startDepth) is an entry
func (t *Tree) Insert(name ndn.Name) (ok bool, oldMd int, newMd int, virtIsEntry bool) {
	nComps := len(name)
	// create node at name and ancestors
	nodes := make([]*Node, nComps+1)
	nodes[0] = t.root
	for i := 1; i <= nComps; i++ {
		parent := nodes[i-1]
		compTlv, _ := tlv.Encode(name[i-1])
		comp := string(compTlv)
		node := parent.children[comp]
		if node != nil {
			nodes[i] = node
			continue
		}

		node = newNode()
		t.nNodes++
		parent.children[comp] = node
		nodes[i] = node

		// store subtree when creating node at NDT prefixLen
		if i == t.ndtPrefixLen {
			ndtIndex := t.getNdtIndex(name)
			prefixV, _ := name[:i].MarshalBinary()
			t.subtrees[ndtIndex][node] = string(prefixV)
		}
	}

	// add entry at name if it does not exist
	if nodes[nComps].IsEntry {
		return false, -1, -1, false
	}
	nodes[nComps].IsEntry = true
	t.nEntries++

	// update maxDepth on nodes
	for i := nComps; i >= 0; i-- {
		node := nodes[i]
		if i == t.startDepth {
			oldMd = node.MaxDepth
		}
		node.updateMaxDepth()
		if i == t.startDepth {
			newMd = node.MaxDepth
			virtIsEntry = node.IsEntry
		}
	}
	return true, oldMd, newMd, virtIsEntry
}

// Erase erases an entry at the specified name.
// Returns:
//   ok: true if deleted, false if entry does not exist
//   oldMd: old MaxDepth at name.GetPrefix(startDepth)
//   newMd: new MaxDepth at name.GetPrefix(startDepth)
//   virtIsEntry: whether node at name.GetPrefix(startDepth) is an entry
func (t *Tree) Erase(name ndn.Name) (ok bool, oldMd int, newMd int, virtIsEntry bool) {
	nComps := len(name)
	// find node at name and ancestors
	nodes := make([]*Node, nComps+1)
	nodes[0] = t.root
	for i := 1; i <= nComps; i++ {
		parent := nodes[i-1]
		compTlv, _ := tlv.Encode(name[i-1])
		comp := string(compTlv)
		nodes[i] = parent.children[comp]
		if nodes[i] == nil {
			return false, -1, -1, false
		}
	}

	// erase entry at name if it exists
	if !nodes[nComps].IsEntry {
		return false, -1, -1, false
	}
	nodes[nComps].IsEntry = false
	t.nEntries--

	// update maxDepth on nodes
	for i := nComps; i >= 0; i-- {
		node := nodes[i]
		if i == t.startDepth {
			oldMd = node.MaxDepth
		}
		node.updateMaxDepth()
		if i == t.startDepth {
			newMd = node.MaxDepth
			virtIsEntry = node.IsEntry
		}

		// keep node if it's root, has entry, or has children
		if i == 0 || node.IsEntry || len(node.children) > 0 {
			continue
		}

		// delete unused node
		if i == t.ndtPrefixLen {
			ndtIndex := t.getNdtIndex(name)
			delete(t.subtrees[ndtIndex], node)
		}
		parent := nodes[i-1]
		compTlv, _ := tlv.Encode(name[i-1])
		delete(parent.children, string(compTlv))
		t.nNodes--
	}
	return true, oldMd, newMd, virtIsEntry
}
