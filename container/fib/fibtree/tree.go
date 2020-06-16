package fibtree

import (
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type GetNdtIndexCallback func(name *ndni.Name) uint64

// FIB tree structure.
type Tree struct {
	startDepth   int
	ndtPrefixLen int
	getNdtIndex  GetNdtIndexCallback

	nEntries int
	nNodes   int
	root     *Node
	subtrees []map[*Node]string // ndtIndex => list of <Node,string(nameV)> tuples, where name.Len()==ndtPrefixLen
}

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

func (t *Tree) CountEntries() int {
	return t.nEntries
}

func (t *Tree) CountNodes() int {
	return t.nNodes
}

// Traversal visitor.
// Returns whether to visit descendants of current node.
type TraverseCallback func(name *ndni.Name, n *Node) bool

// Traverse entire tree.
func (t *Tree) Traverse(cb TraverseCallback) {
	t.root.traverse("", cb)
}

// Traverse subtrees of a specified ndtIndex.
func (t *Tree) TraverseSubtree(ndtIndex uint64, cb TraverseCallback) {
	for n, nameV := range t.subtrees[ndtIndex] {
		n.traverse(nameV, cb)
	}
}

// Insert entry at name.
// Returns:
//   ok: true if inserted, false if entry already exists
//   oldMd: old MaxDepth at name.GetPrefix(startDepth)
//   newMd: new MaxDepth at name.GetPrefix(startDepth)
//   virtIsEntry: whether node at name.GetPrefix(startDepth) is an entry
func (t *Tree) Insert(name *ndni.Name) (ok bool, oldMd int, newMd int, virtIsEntry bool) {
	nComps := name.Len()
	// create node at name and ancestors
	nodes := make([]*Node, nComps+1)
	nodes[0] = t.root
	for i := 1; i <= nComps; i++ {
		parent := nodes[i-1]
		comp := string(name.GetComp(i - 1))
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
			t.subtrees[ndtIndex][node] = string(name.GetPrefix(i).GetValue())
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

// Erase entry at name.
// Returns:
//   ok: true if deleted, false if entry does not exist
//   oldMd: old MaxDepth at name.GetPrefix(startDepth)
//   newMd: new MaxDepth at name.GetPrefix(startDepth)
//   virtIsEntry: whether node at name.GetPrefix(startDepth) is an entry
func (t *Tree) Erase(name *ndni.Name) (ok bool, oldMd int, newMd int, virtIsEntry bool) {
	nComps := name.Len()
	// find node at name and ancestors
	nodes := make([]*Node, nComps+1)
	nodes[0] = t.root
	for i := 1; i <= nComps; i++ {
		parent := nodes[i-1]
		comp := string(name.GetComp(i - 1))
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
		delete(parent.children, string(name.GetComp(i-1)))
		t.nNodes--
	}
	return true, oldMd, newMd, virtIsEntry
}
