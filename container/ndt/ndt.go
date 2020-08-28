package ndt

import (
	"math/rand"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Entry contains information from an NDT entry.
type Entry struct {
	Index int    `json:"index"`
	Value int    `json:"value"`
	Hits  uint32 `json:"hits"`
}

// Ndt represents a Name Dispatch Table (NDT).
type Ndt struct {
	cfg      Config
	replicas map[eal.NumaSocket]*replica
	threads  []*Thread
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

// Threads returns lookup threads.
func (ndt *Ndt) Threads() (list []*Thread) {
	return ndt.threads
}

// Close releases memory of all replicas and threads.
func (ndt *Ndt) Close() error {
	for _, ndtt := range ndt.threads {
		ndtt.Close()
	}
	ndt.threads = nil
	for _, ndtr := range ndt.replicas {
		ndtr.Close()
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
	entry = ndt.firstReplica().Read(int(index))
	for _, ndtt := range ndt.Threads() {
		entry.Hits += ndtt.hitCounters(ndt.cfg.Capacity)[index]
	}
	return entry
}

// List returns all entries.
func (ndt *Ndt) List() (list []Entry) {
	list = make([]Entry, ndt.cfg.Capacity)
	ndtr := ndt.firstReplica()
	for i := range list {
		list[i] = ndtr.Read(i)
	}

	for _, ndtt := range ndt.Threads() {
		for index, hit := range ndtt.hitCounters(ndt.cfg.Capacity) {
			list[index].Hits += hit
		}
	}
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
func (ndt *Ndt) Randomize(max int) {
	for i := 0; i < ndt.cfg.Capacity; i++ {
		ndt.Update(uint64(i), uint8(rand.Intn(max)))
	}
}

// Lookup queries a name without incrementing hit counters.
func (ndt *Ndt) Lookup(name ndn.Name) (index uint64, value uint8) {
	return ndt.firstReplica().Lookup(name)
}

// New creates an Ndt.
// sockets indicates NUMA sockets of lookup threads.
func New(cfg Config, sockets []eal.NumaSocket) (ndt *Ndt) {
	cfg.applyDefaults()
	ndt = &Ndt{
		cfg:      cfg,
		replicas: make(map[eal.NumaSocket]*replica),
		threads:  make([]*Thread, len(sockets)),
	}

	for i, socket := range sockets {
		if ndt.replicas[socket] == nil {
			ndt.replicas[socket] = newReplica(cfg, socket)
		}
		ndt.threads[i] = newThread(ndt, socket)
	}
	return ndt
}
