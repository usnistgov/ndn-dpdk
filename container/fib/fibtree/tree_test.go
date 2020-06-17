package fibtree_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibtree"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

var makeAR = testenv.MakeAR

func makeTree() *fibtree.Tree {
	return fibtree.New(2, 1, 16, func(name ndn.Name) uint64 {
		return uint64(name[0].Value[0] & 0x0F)
	})
}

func TestInsertErase(testingT *testing.T) {
	assert, _ := makeAR(testingT)
	t := makeTree()
	assert.Equal(0, t.CountEntries())
	assert.Equal(1, t.CountNodes())

	ok, oldMd, newMd, virtIsEntry := t.Insert(ndn.ParseName("/%00/A/B/B"))
	assert.True(ok)
	assert.Equal(0, oldMd)
	assert.Equal(2, newMd)
	assert.False(virtIsEntry)
	assert.Equal(1, t.CountEntries())
	assert.Equal(5, t.CountNodes())

	ok, oldMd, newMd, virtIsEntry = t.Insert(ndn.ParseName("/%00/A/B/B"))
	assert.False(ok)
	assert.Equal(1, t.CountEntries())
	assert.Equal(5, t.CountNodes())

	ok, oldMd, newMd, virtIsEntry = t.Insert(ndn.ParseName("/%00/A/C"))
	assert.True(ok)
	assert.Equal(2, oldMd)
	assert.Equal(2, newMd)
	assert.False(virtIsEntry)
	assert.Equal(2, t.CountEntries())
	assert.Equal(6, t.CountNodes())

	ok, oldMd, newMd, virtIsEntry = t.Erase(ndn.ParseName("/%00/A/B/B"))
	assert.True(ok)
	assert.Equal(2, oldMd)
	assert.Equal(1, newMd)
	assert.False(virtIsEntry)
	assert.Equal(1, t.CountEntries())
	assert.Equal(4, t.CountNodes())

	ok, oldMd, newMd, virtIsEntry = t.Insert(ndn.ParseName("/%00/A"))
	assert.True(ok)
	assert.Equal(1, oldMd)
	assert.Equal(1, newMd)
	assert.True(virtIsEntry)
	assert.Equal(2, t.CountEntries())
	assert.Equal(4, t.CountNodes())

	ok, oldMd, newMd, virtIsEntry = t.Erase(ndn.ParseName("/%00/A/B/B"))
	assert.False(ok)
	assert.Equal(2, t.CountEntries())
	assert.Equal(4, t.CountNodes())

	ok, oldMd, newMd, virtIsEntry = t.Erase(ndn.ParseName("/%00"))
	assert.False(ok)

	ok, oldMd, newMd, virtIsEntry = t.Erase(ndn.ParseName("/%00/A/C"))
	assert.True(ok)
	assert.Equal(1, oldMd)
	assert.Equal(0, newMd)
	assert.True(virtIsEntry)
	assert.Equal(1, t.CountEntries())
	assert.Equal(3, t.CountNodes())

	ok, oldMd, newMd, virtIsEntry = t.Erase(ndn.ParseName("/%00/A"))
	assert.True(ok)
	assert.Equal(0, oldMd)
	assert.Equal(0, newMd)
	assert.False(virtIsEntry)
	assert.Equal(0, t.CountEntries())
	assert.Equal(1, t.CountNodes())
}

type visitor struct {
	nodes    []string
	entries  []string
	MaxDepth map[string]int
}

func newVisitor() (v *visitor) {
	v = new(visitor)
	v.MaxDepth = make(map[string]int)
	return v
}

func (v *visitor) TraverseCallback(name ndn.Name, n *fibtree.Node) bool {
	uri := name.String()
	letter := uri[len(uri)-1:]
	v.nodes = append(v.nodes, letter)
	if n.IsEntry {
		v.entries = append(v.entries, letter)
	}
	v.MaxDepth[letter] = n.MaxDepth
	return len(name) < 5
}

func (v *visitor) Nodes() string {
	sort.Strings(v.nodes)
	return strings.Join(v.nodes, "")
}

func (v *visitor) Entries() string {
	sort.Strings(v.entries)
	return strings.Join(v.entries, "")
}

func TestTraverse(testingT *testing.T) {
	assert, _ := makeAR(testingT)
	t := makeTree()

	t.Insert(ndn.ParseName("/%00/A"))
	t.Insert(ndn.ParseName("/%00/A/B"))
	t.Insert(ndn.ParseName("/%00/A/B/C"))
	t.Insert(ndn.ParseName("/%00/A/B/C/D/E/F/G"))
	t.Insert(ndn.ParseName("/%00/A/H"))
	t.Insert(ndn.ParseName("/%00/I"))
	t.Insert(ndn.ParseName("/%01/J"))
	t.Insert(ndn.ParseName("/%01/J/K/L"))
	t.Insert(ndn.ParseName("/%01/M"))
	t.Insert(ndn.ParseName("/%04/N/O"))
	t.Insert(ndn.ParseName("/%07/P/Q/R"))

	v := newVisitor()
	t.Traverse(v.TraverseCallback)
	assert.Equal("ABCHIJLMOR", v.Entries())
	assert.Equal("/0147ABCDHIJKLMNOPQR", v.Nodes())
	assert.Equal(6, v.MaxDepth["A"])
	assert.Equal(0, v.MaxDepth["I"])
	assert.Equal(1, v.MaxDepth["N"])
	assert.Equal(2, v.MaxDepth["P"])

	v = newVisitor()
	t.TraverseSubtree(0, v.TraverseCallback)
	assert.Equal("ABCHI", v.Entries())
	assert.Equal("0ABCDHI", v.Nodes())
	assert.Contains(v.MaxDepth, "A")
	assert.Equal(6, v.MaxDepth["A"])
	assert.Contains(v.MaxDepth, "I")
	assert.Equal(0, v.MaxDepth["I"])
	assert.NotContains(v.MaxDepth, "J")

	v = newVisitor()
	t.TraverseSubtree(2, v.TraverseCallback)
	assert.Equal("", v.Entries())
	assert.Equal("", v.Nodes())
	assert.Empty(v.MaxDepth)
}
