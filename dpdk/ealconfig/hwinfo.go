package ealconfig

import (
	"github.com/jaypipes/ghw"
	"github.com/pkg/math"
)

// CoreInfo describes a CPU core.
type CoreInfo struct {
	ID          int
	NumaSocket  int
	HyperThread bool
}

// HwInfoSource provides information about hardware.
type HwInfoSource interface {
	// Cores provides information about CPU cores.
	Cores() []CoreInfo
}

// DefaultHwInfoSource returns the default HwInfoSource implementation.
func DefaultHwInfoSource() HwInfoSource {
	return defaultHwInfoSource{}
}

type defaultHwInfoSource struct{}

func (defaultHwInfoSource) Cores() (list []CoreInfo) {
	topo, e := ghw.Topology()
	if e != nil {
		log.WithError(e).Panic("ghw.Topology")
	}

	for _, node := range topo.Nodes {
		for _, core := range node.Cores {
			ht := false
			for _, lp := range core.LogicalProcessors {
				list = append(list, CoreInfo{
					ID:          lp,
					NumaSocket:  node.ID,
					HyperThread: ht,
				})
				ht = true
			}
		}
	}
	return list
}

func maxNumaSocket(hwInfo HwInfoSource) (maxSocket int) {
	cores := hwInfo.Cores()
	for _, core := range cores {
		maxSocket = math.MaxInt(maxSocket, core.NumaSocket)
	}
	return maxSocket
}

type hwInfoLCores struct {
	SocketSet      map[int]bool
	CoreSet        map[int]bool
	PrimaryCores   map[int][]int
	SecondaryCores map[int][]int
}

func gatherHwInfoLCores(hwInfo HwInfoSource) (info hwInfoLCores) {
	info = hwInfoLCores{
		SocketSet:      make(map[int]bool),
		CoreSet:        make(map[int]bool),
		PrimaryCores:   make(map[int][]int),
		SecondaryCores: make(map[int][]int),
	}
	for _, core := range hwInfo.Cores() {
		info.SocketSet[core.NumaSocket] = true
		info.CoreSet[core.ID] = true
		m := &info.PrimaryCores
		if core.HyperThread {
			m = &info.SecondaryCores
		}
		(*m)[core.NumaSocket] = append((*m)[core.NumaSocket], core.ID)
	}
	return info
}
