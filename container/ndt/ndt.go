// Package ndt implements the Name Dispatch Table.
package ndt

import (
	"math/rand"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/zyedidia/generic/mapset"
)

// Entry contains information from an NDT entry.
type Entry struct {
	Index uint64 `json:"index" gqldesc:"Entry index."`
	Value uint8  `json:"value" gqldesc:"Entry value, i.e. forwarding thread index."`
	Hits  uint32 `json:"hits" gqldesc:"Hit counter value, uint32 wraparound."`
}

// Ndt represents a Name Dispatch Table (NDT).
type Ndt struct {
	cfg      Config
	replicas map[eal.NumaSocket]*replica
	queriers mapset.Set[*Querier]
}

// Config returns effective configuration.
func (ndt *Ndt) Config() Config {
	return ndt.cfg
}

func (ndt *Ndt) firstReplica() *replica {
	for _, ndtr := range ndt.replicas {
		return ndtr
	}
	panic("NDT has no replica")
}

func (ndt *Ndt) getReplica(socket eal.NumaSocket) *replica {
	if ndtr, ok := ndt.replicas[socket]; ok {
		return ndtr
	}
	return ndt.firstReplica()
}

// Close releases memory of all replicas and threads.
func (ndt *Ndt) Close() error {
	ndt.queriers.Each(func(ndq *Querier) {
		ndq.Clear(ndt)
	})
	ndt.queriers.Clear()

	for _, ndtr := range ndt.replicas {
		eal.Free(ndtr)
	}
	ndt.replicas = nil
	return nil
}

// ComputeHash computes the hash used for a name.
func (ndt *Ndt) ComputeHash(name ndn.Name) uint64 {
	if len(name) > ndt.cfg.PrefixLen {
		name = name[:ndt.cfg.PrefixLen]
	}
	pname := ndni.NewPName(name)
	defer pname.Free()
	return pname.ComputeHash()
}

// IndexOfHash returns table index used for a hash.
func (ndt *Ndt) IndexOfHash(hash uint64) uint64 {
	return hash & ndt.cfg.indexMask()
}

// IndexOfName returns table index used for a name.
func (ndt *Ndt) IndexOfName(name ndn.Name) uint64 {
	return ndt.IndexOfHash(ndt.ComputeHash(name))
}

// Get returns one entry.
func (ndt *Ndt) Get(index uint64) (entry Entry) {
	entry = ndt.firstReplica().Read(index)
	ndt.queriers.Each(func(ndq *Querier) {
		entry.Hits += ndq.hitCounters(ndt.cfg.Capacity)[index]
	})
	return entry
}

// List returns all entries.
func (ndt *Ndt) List() (list []Entry) {
	list = make([]Entry, ndt.cfg.Capacity)
	ndtr := ndt.firstReplica()
	for i := uint64(0); i < uint64(ndt.cfg.Capacity); i++ {
		list[i] = ndtr.Read(i)
	}

	ndt.queriers.Each(func(ndq *Querier) {
		for index, hit := range ndq.hitCounters(ndt.cfg.Capacity) {
			list[index].Hits += hit
		}
	})
	return list
}

// Update updates an element.
func (ndt *Ndt) Update(index uint64, value uint8) {
	for _, ndtr := range ndt.replicas {
		ndtr.Update(index, value)
	}
}

// Randomize updates all elements to random values < max.
// This should only be used during initialization.
func (ndt *Ndt) Randomize(max uint8) {
	for i := 0; i < ndt.cfg.Capacity; i++ {
		ndt.Update(uint64(i), uint8(rand.Intn(int(max))))
	}
}

// Lookup queries a name without incrementing hit counters.
func (ndt *Ndt) Lookup(name ndn.Name) (index uint64, value uint8) {
	return ndt.firstReplica().Lookup(name)
}

// New creates an NDT.
// sockets are NUMA sockets where NDT replicas are needed; duplicates are permitted.
func New(cfg Config, sockets []eal.NumaSocket) (ndt *Ndt) {
	cfg.applyDefaults()
	ndt = &Ndt{
		cfg:      cfg,
		replicas: map[eal.NumaSocket]*replica{},
		queriers: mapset.New[*Querier](),
	}

	if len(sockets) == 0 {
		sockets = []eal.NumaSocket{{}}
	}
	for _, socket := range sockets {
		socket = eal.RewriteAnyNumaSocketFirst.Rewrite(socket)
		if ndt.replicas[socket] == nil {
			ndt.replicas[socket] = newReplica(cfg, socket)
		}
	}
	return ndt
}
