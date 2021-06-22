package hwinfo

import (
	"github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/logging"
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

// MaxNumaSocket determines the maximum NUMA socket.
func (cores Cores) MaxNumaSocket() int {
	maxSocket := -1
	for _, core := range cores {
		maxSocket = math.MaxInt(maxSocket, core.NumaSocket)
	}
	return maxSocket
}

// HasLogicalCore determines whether a logical core exists.
func (cores Cores) HasLogicalCore(id int) bool {
	for _, core := range cores {
		if core.LogicalCore == id {
			return true
		}
	}
	return false
}

// ListPrimary returns a list of logical cores that are the first logical core in each physical core.
// If socket is non-negative, the list only includes logical cores on this NUMA socket.
func (cores Cores) ListPrimary(socket int) []int {
	return cores.listHyperThread(socket, false)
}

// ListSecondary returns a list of logical cores that are not in ListPrimary().
// If socket is non-negative, the list only includes logical cores on this NUMA socket.
func (cores Cores) ListSecondary(socket int) []int {
	return cores.listHyperThread(socket, true)
}

func (cores Cores) listHyperThread(socket int, secondary bool) (list []int) {
	ht := map[[2]int]bool{}
	for _, core := range cores {
		if socket >= 0 && core.NumaSocket != socket {
			continue
		}
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
