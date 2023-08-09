// Package hwinfo gathers hardware information.
package hwinfo

import (
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/zyedidia/generic/mapset"
)

var logger = logging.New("hwinfo")

// CoreInfo describes a logical CPU core.
type CoreInfo struct {
	ID          int // logical core ID
	NumaSocket  int // NUMA socket number
	PhysicalKey int // physical core key, same value for hyper-threads on a physical core
}

// Cores contains information about CPU cores.
type Cores []CoreInfo

// ByNumaSocket classifies cores as map[NumaSocket]Cores.
func (cores Cores) ByNumaSocket() (m map[int]Cores) {
	m = map[int]Cores{}
	for _, core := range cores {
		m[core.NumaSocket] = append(m[core.NumaSocket], core)
	}
	return m
}

// MaxNumaSocket determines the maximum NUMA socket.
func (cores Cores) MaxNumaSocket() int {
	maxSocket := -1
	for _, core := range cores {
		maxSocket = max(maxSocket, core.NumaSocket)
	}
	return maxSocket
}

// ByID converts to map[ID]CoreInfo.
func (cores Cores) ByID() (m map[int]CoreInfo) {
	m = map[int]CoreInfo{}
	for _, core := range cores {
		m[core.ID] = core
	}
	return m
}

// ListPrimary returns a list of logical cores that are the first logical core in each physical core.
func (cores Cores) ListPrimary() []int {
	return cores.listHyperThread(false)
}

// ListSecondary returns a list of logical cores that are not in ListPrimary().
func (cores Cores) ListSecondary() []int {
	return cores.listHyperThread(true)
}

func (cores Cores) listHyperThread(secondary bool) (list []int) {
	physicalSet := mapset.New[int]()
	for _, core := range cores {
		if physicalSet.Has(core.PhysicalKey) == secondary {
			list = append(list, core.ID)
		}
		physicalSet.Put(core.PhysicalKey)
	}
	return list
}

// Provider provides information about hardware.
type Provider interface {
	// Cores provides information about CPU cores.
	Cores() Cores
}

// Default is the default Provider implementation.
var Default Provider = &procinfoProvider{}
