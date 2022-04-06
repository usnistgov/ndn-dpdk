// Package hwinfo gathers hardware information.
package hwinfo

import (
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/zyedidia/generic"
)

var logger = logging.New("hwinfo")

// CoreInfo describes a logical CPU core.
type CoreInfo struct {
	NumaSocket   int
	PhysicalCore int
	LogicalCore  int
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
		maxSocket = generic.Max(maxSocket, core.NumaSocket)
	}
	return maxSocket
}

// ByLogicalCore converts to map[LogicalCore]CoreInfo.
func (cores Cores) ByLogicalCore() (m map[int]CoreInfo) {
	m = map[int]CoreInfo{}
	for _, core := range cores {
		m[core.LogicalCore] = core
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
	ht := map[[2]int]bool{}
	for _, core := range cores {
		key := [2]int{core.NumaSocket, core.PhysicalCore}
		if ht[key] == secondary {
			list = append(list, core.LogicalCore)
		}
		ht[key] = true
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
