package dpdk

/*
#include "../core/common.h"
*/
import "C"
import (
	"sort"
)

// An interface to provide information about LCores.
// Mock of this interface allows unit testing of LCoreAllocator.
type lCoreProvider interface {
	ListSlaves() []LCore
	IsBusy(lc LCore) bool
	GetNumaSocket(lc LCore) NumaSocket
}

type ealLCoreProvider struct{}

func (ealLCoreProvider) ListSlaves() []LCore {
	return ListSlaveLCores()
}

func (ealLCoreProvider) IsBusy(lc LCore) bool {
	return lc.IsBusy()
}

func (ealLCoreProvider) GetNumaSocket(lc LCore) NumaSocket {
	return lc.GetNumaSocket()
}

// LCoreAllocRoleConfig contains lcore allocation config for a role.
type LCoreAllocRoleConfig struct {
	// List of LCores reserved for this role.
	LCores []int
	// Number of LCores per NUMA socket reserved for this role.
	// Key '-1' indicates for each NUMA socket.
	PerNuma map[int]int
}

func (roleCfg LCoreAllocRoleConfig) getLimitOn(numaSocket NumaSocket) int {
	if roleCfg.PerNuma == nil {
		return 0
	}
	if n, ok := roleCfg.PerNuma[numaSocket.ID()]; ok {
		return n
	}
	return roleCfg.PerNuma[-1]
}

// LCoreAllocConfig contains per-role lcore allocation config.
type LCoreAllocConfig map[string]LCoreAllocRoleConfig

// LCoreAllocator allocates lcores to roles.
type LCoreAllocator struct {
	Provider  lCoreProvider
	Config    LCoreAllocConfig
	allocated [C.RTE_MAX_LCORE]string
}

type lCorePredicate func(lc LCore) bool

func (la *LCoreAllocator) invert(pred lCorePredicate) lCorePredicate {
	return func(lc LCore) bool {
		return !pred(lc)
	}
}

func (la *LCoreAllocator) lcIsIdle() lCorePredicate {
	return func(lc LCore) bool {
		return !la.Provider.IsBusy(lc)
	}
}

func (la *LCoreAllocator) lcIsAvailable() lCorePredicate {
	return func(lc LCore) bool {
		return la.allocated[lc.ID()] == "" && !la.Provider.IsBusy(lc)
	}
}

func (la *LCoreAllocator) lcOnNuma(numaSocket NumaSocket) lCorePredicate {
	return func(lc LCore) bool {
		return numaSocket.IsAny() || la.Provider.GetNumaSocket(lc).ID() == numaSocket.ID()
	}
}

func (la *LCoreAllocator) lcInList(list []int) lCorePredicate {
	sorted := append([]int{}, list...)
	sort.Ints(sorted)

	return func(lc LCore) bool {
		i := sort.SearchInts(sorted, lc.ID())
		return i < len(sorted) && sorted[i] == lc.ID()
	}
}

func (la *LCoreAllocator) lcAllocatedTo(role string) lCorePredicate {
	return func(lc LCore) bool {
		return la.allocated[lc.ID()] == role
	}
}

// Return subset of lcores that match all predicates.
func (la *LCoreAllocator) filter(lcores []LCore, predicates ...lCorePredicate) (filtered []LCore) {
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
func (la *LCoreAllocator) classifyByNuma(lcores []LCore) (m map[NumaSocket][]LCore) {
	m = make(map[NumaSocket][]LCore)
	for _, lc := range lcores {
		numaSocket := la.Provider.GetNumaSocket(lc)
		m[numaSocket] = append(m[numaSocket], lc)
	}
	return m
}

func (la *LCoreAllocator) pick(role string, numaSocket NumaSocket) LCore {
	lcores := la.Provider.ListSlaves()
	avails := la.filter(lcores, la.lcIsAvailable())
	if len(avails) == 0 {
		return LCore{}
	}
	numaAvails := la.filter(avails, la.lcOnNuma(numaSocket))

	// 0. When Config is empty, satisfy every request.
	if len(la.Config) == 0 {
		// (1) Allocate from requested NumaSocket.
		if !numaSocket.IsAny() && len(numaAvails) > 0 {
			return numaAvails[0]
		}
		// (2) Allocate from least occupied NumaSocket.
		availsByNuma := la.classifyByNuma(avails)
		var candidate LCore
		candidateRem := 0
		for _, availsOnNuma := range availsByNuma {
			// (4) Prefer the NumaSocket with most unreserved lcores.
			if len(availsOnNuma) > 0 && len(availsOnNuma) > candidateRem {
				candidate = availsOnNuma[0]
				candidateRem = len(availsOnNuma)
			}
		}
		return candidate
	}

	roleCfg := la.Config[role]
	// 1. Allocate from roleCfg.LCores on numaSocket.
	numaCfgLCores := la.filter(numaAvails, la.lcInList(roleCfg.LCores))
	if len(numaCfgLCores) > 0 {
		return numaCfgLCores[0]
	}

	// 2. Allocate on numaSocket within roleCfg.PerNuma limit.
	// (1) Find LCores on numaSocket unreserved by other roles.
	var unreservedPred []lCorePredicate
	for otherRole, otherRoleCfg := range la.Config {
		if otherRole != role {
			unreservedPred = append(unreservedPred, la.invert(la.lcInList(otherRoleCfg.LCores)))
		}
	}
	numaUnreserved := la.filter(numaAvails, unreservedPred...)
	// (2) Determine how many LCores on numaSocket is used by role.
	numaLCores := la.filter(lcores, la.lcOnNuma(numaSocket))
	nNumaAllocated := len(la.filter(numaLCores, la.lcAllocatedTo(role)))
	// (3) Allocate if nNumaAllocated is less than roleCfg.PerNuma[numaSocket].
	if nNumaAllocated < roleCfg.getLimitOn(numaSocket) && len(numaUnreserved) > 0 {
		return numaUnreserved[0]
	}

	// 3. Allocate from roleCfg.LCores on other NumaSocket.
	remoteAvails := la.filter(avails, la.invert(la.lcOnNuma(numaSocket)))
	remoteCfgLCores := la.filter(remoteAvails, la.lcInList(roleCfg.LCores))
	if len(remoteCfgLCores) > 0 {
		return remoteCfgLCores[0]
	}

	// 4. Allocate on other NumaSocket within roleCfg.PerNuma limit.
	// (1) Find LCores on other NumaSockets unreserved by other roles.
	remoteUnreservedByNuma := la.classifyByNuma(la.filter(remoteAvails, unreservedPred...))
	var candidate LCore
	candidateRem := 0
	for remoteSocket, remoteUnreserved := range remoteUnreservedByNuma {
		// (2) Determine how many LCores on remoteSocket is used by role.
		remoteLCores := la.filter(lcores, la.lcOnNuma(remoteSocket))
		nRemoteAllocated := len(la.filter(remoteLCores, la.lcAllocatedTo(role)))
		// (3) Proceed only if nRemoteAllocated is less than roleCfg.PerNuma[remoteSocket].
		if nRemoteAllocated >= roleCfg.getLimitOn(remoteSocket) {
			continue
		}
		// (4) Prefer the NumaSocket with most unreserved lcores.
		if len(remoteUnreserved) > 0 && len(remoteUnreserved) > candidateRem {
			candidate = remoteUnreserved[0]
			candidateRem = len(remoteUnreserved)
		}
	}
	if candidate.IsValid() {
		return candidate
	}

	// 4. Fail.
	return LCore{}
}

// Alloc allocates an lcore for a role.
func (la *LCoreAllocator) Alloc(role string, numaSocket NumaSocket) (lc LCore) {
	lc = la.pick(role, numaSocket)
	if !lc.IsValid() {
		return lc
	}

	la.allocated[lc.ID()] = role
	log.WithFields(makeLogFields("role", role, "socket", numaSocket,
		"lc", lc, "lc-socket", la.Provider.GetNumaSocket(lc))).Info("lcore allocated")
	return lc
}

// Free release an allocated lcore.
func (la *LCoreAllocator) Free(lc LCore) {
	if la.allocated[lc.ID()] == "" {
		panic("lcore double free")
	}
	log.WithFields(makeLogFields("lc", lc, "role", la.allocated[lc.ID()], "socket", la.Provider.GetNumaSocket(lc))).Info("lcore freed")
	la.allocated[lc.ID()] = ""
}

// Clear deletes all allocations.
func (la *LCoreAllocator) Clear() {
	for lc, role := range la.allocated {
		if role != "" {
			la.Free(LCoreFromID(lc))
		}
	}
}

// LCoreAlloc is a global instance of LCoreAlloc using EAL provider.
var LCoreAlloc LCoreAllocator

func init() {
	LCoreAlloc.Provider = ealLCoreProvider{}
	LCoreAlloc.Config = make(LCoreAllocConfig)
}
