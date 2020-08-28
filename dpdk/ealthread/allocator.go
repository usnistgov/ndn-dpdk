package ealthread

import (
	"sort"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// AllocConfig contains per-role lcore allocation config.
type AllocConfig map[string]AllocRoleConfig

// AllocRoleConfig contains lcore allocation config for a role.
type AllocRoleConfig struct {
	// List of lcores reserved for this role.
	LCores []int
	// Number of lcores on a specified NUMA socket.
	OnNuma map[int]int
	// Number of lcores on each NUMA socket.
	EachNuma int
}

func (c AllocRoleConfig) limitOn(socket eal.NumaSocket) int {
	if c.OnNuma == nil {
		return c.EachNuma
	}
	if n, ok := c.OnNuma[socket.ID()]; ok {
		return n
	}
	return c.EachNuma
}

// Allocator allocates lcores to roles.
type Allocator struct {
	Config    AllocConfig
	provider  lCoreProvider
	allocated [eal.MaxLCoreID + 1]string
}

// NewAllocator creates an Allocator.
func NewAllocator(provider lCoreProvider) *Allocator {
	return &Allocator{
		Config:   make(AllocConfig),
		provider: provider,
	}
}

type lCorePredicate func(lc eal.LCore) bool

func (la *Allocator) invert(pred lCorePredicate) lCorePredicate {
	return func(lc eal.LCore) bool {
		return !pred(lc)
	}
}

func (la *Allocator) lcIsIdle() lCorePredicate {
	return func(lc eal.LCore) bool {
		return !la.provider.IsBusy(lc)
	}
}

func (la *Allocator) lcIsAvailable() lCorePredicate {
	return func(lc eal.LCore) bool {
		return la.allocated[lc.ID()] == "" && !la.provider.IsBusy(lc)
	}
}

func (la *Allocator) lcOnNuma(socket eal.NumaSocket) lCorePredicate {
	return func(lc eal.LCore) bool {
		return socket.IsAny() || la.provider.NumaSocketOf(lc).ID() == socket.ID()
	}
}

func (la *Allocator) lcInList(list []int) lCorePredicate {
	sorted := append([]int{}, list...)
	sort.Ints(sorted)

	return func(lc eal.LCore) bool {
		i := sort.SearchInts(sorted, lc.ID())
		return i < len(sorted) && sorted[i] == lc.ID()
	}
}

func (la *Allocator) lcAllocatedTo(role string) lCorePredicate {
	return func(lc eal.LCore) bool {
		return la.allocated[lc.ID()] == role
	}
}

// Return subset of lcores that match all predicates.
func (la *Allocator) filter(lcores []eal.LCore, predicates ...lCorePredicate) (filtered []eal.LCore) {
L:
	for _, lc := range lcores {
		for _, pred := range predicates {
			if !pred(lc) {
				continue L
			}
		}
		filtered = append(filtered, lc)
	}
	return filtered
}

// Classify lcores by NumaSocket.
func (la *Allocator) classifyByNuma(lcores []eal.LCore) (m map[eal.NumaSocket][]eal.LCore) {
	m = make(map[eal.NumaSocket][]eal.LCore)
	for _, lc := range lcores {
		socket := la.provider.NumaSocketOf(lc)
		m[socket] = append(m[socket], lc)
	}
	return m
}

func (la *Allocator) pick(role string, socket eal.NumaSocket) eal.LCore {
	// 0. When Config is empty, satisfy every request.
	if len(la.Config) == 0 {
		return la.pickNoConfig(role, socket)
	}

	// 1. Allocate on preferred NUMA socket.
	if lcores := la.pickCfgOnNuma(role, socket); len(lcores) > 0 {
		return lcores[0]
	}

	// 2. Allocate on other NUMA sockets.
	numaLCores := la.classifyByNuma(la.provider.Workers())
	for remoteSocket := range numaLCores {
		numaLCores[remoteSocket] = la.pickCfgOnNuma(role, remoteSocket)
	}
	return la.pickLeastOccupied(numaLCores)
}

func (la *Allocator) pickNoConfig(role string, socket eal.NumaSocket) eal.LCore {
	workers := la.provider.Workers()
	avails := la.filter(workers, la.lcIsAvailable())

	// 1. Allocate from preferred NUMA socket.
	if !socket.IsAny() {
		if numaAvails := la.filter(avails, la.lcOnNuma(socket)); len(numaAvails) > 0 {
			return numaAvails[0]
		}
	}

	// 2. Allocate from least occupied NUMA socket.
	availsByNuma := la.classifyByNuma(avails)
	return la.pickLeastOccupied(availsByNuma)
}

func (la *Allocator) pickCfgOnNuma(role string, socket eal.NumaSocket) []eal.LCore {
	workers := la.provider.Workers()
	avails := la.filter(workers, la.lcIsAvailable(), la.lcOnNuma(socket))
	rc := la.Config[role]

	// 1. Allocate from role-specific list on specified NUMA socket.
	if listed := la.filter(avails, la.lcInList(rc.LCores)); len(listed) > 0 {
		return listed
	}

	// 2. Allocate within role-specific per-socket limit on specified NUMA socket.
	// (1) Find lcores not listed by other roles.
	var unreservedPred []lCorePredicate
	for otherRole, otherRc := range la.Config {
		if otherRole != role {
			unreservedPred = append(unreservedPred, la.invert(la.lcInList(otherRc.LCores)))
		}
	}
	unreserved := la.filter(avails, unreservedPred...)

	// (2) Count lcores already allocated to this role.
	nAllocated := len(la.filter(workers, la.lcOnNuma(socket), la.lcAllocatedTo(role)))

	// (3) Allocate if within limit.
	if nAllocated < rc.limitOn(socket) && len(unreserved) > 0 {
		return unreserved
	}

	return nil
}

func (la *Allocator) pickLeastOccupied(availsByNuma map[eal.NumaSocket][]eal.LCore) eal.LCore {
	var candidate eal.LCore
	candidateRem := 0
	for _, numaAvails := range availsByNuma {
		if len(numaAvails) > candidateRem {
			candidate = numaAvails[0]
			candidateRem = len(numaAvails)
		}
	}
	return candidate
}

// Alloc allocates an lcore for a role.
func (la *Allocator) Alloc(role string, socket eal.NumaSocket) (lc eal.LCore) {
	lc = la.pick(role, socket)
	if !lc.Valid() {
		return lc
	}

	la.allocated[lc.ID()] = role
	log.WithFields(makeLogFields("role", role, "socket", socket,
		"lc", lc, "lc-socket", la.provider.NumaSocketOf(lc))).Info("lcore allocated")
	return lc
}

// AllocGroup allocates several LCores for each NUMA socket.
// Returns a list organized by roles, or nil on failure.
func (la *Allocator) AllocGroup(roles []string, sockets []eal.NumaSocket) (list [][]eal.LCore) {
	list = make([][]eal.LCore, len(roles))
	for _, socket := range sockets {
		for i, role := range roles {
			lc := la.pick(role, socket)
			if !lc.Valid() {
				goto FAIL
			}
			la.allocated[lc.ID()] = role
			list[i] = append(list[i], lc)
		}
	}
	log.WithFields(makeLogFields("roles", roles, "sockets", sockets, "list", list)).Info("lcores allocated")
	return list

FAIL:
	for _, roleList := range list {
		for _, lc := range roleList {
			la.allocated[lc.ID()] = ""
		}
	}
	return nil
}

// AllocMax allocates all remaining LCores to a role.
func (la *Allocator) AllocMax(role string) (list []eal.LCore) {
	for {
		if lc := la.Alloc(role, eal.NumaSocket{}); lc.Valid() {
			list = append(list, lc)
		} else {
			break
		}
	}
	return list
}

// Free deallocates an lcore.
func (la *Allocator) Free(lc eal.LCore) {
	if la.allocated[lc.ID()] == "" {
		panic("lcore double free")
	}
	log.WithFields(makeLogFields("lc", lc, "role", la.allocated[lc.ID()], "socket", la.provider.NumaSocketOf(lc))).Info("lcore freed")
	la.allocated[lc.ID()] = ""
}

// Clear deletes all allocations.
func (la *Allocator) Clear() {
	for lc, role := range la.allocated {
		if role != "" {
			la.Free(eal.LCoreFromID(lc))
		}
	}
}

// DefaultAllocator is the default instance of Allocator.
var DefaultAllocator = NewAllocator(ealLCoreProvider{})
