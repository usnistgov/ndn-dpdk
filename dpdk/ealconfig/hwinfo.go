package ealconfig

import (
	"os"
	"path"
	"strings"

	"github.com/fromanirh/cpuset"
	"github.com/jaypipes/ghw"
	"github.com/pkg/math"
	"go.uber.org/zap"
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

func (hwInfo defaultHwInfoSource) Cores() (list []CoreInfo) {
	topo := hwInfo.getTopo()
	cpusetFilter := hwInfo.getCpusetFilter()

	for _, node := range topo.Nodes {
		for _, core := range node.Cores {
			ht := false
			for _, lp := range core.LogicalProcessors {
				if !cpusetFilter(lp) {
					continue
				}
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

func (defaultHwInfoSource) getTopo() *ghw.TopologyInfo {
	topo, e := ghw.Topology()
	if e != nil {
		logger.Panic("ghw.Topology",
			zap.Error(e),
		)
	}
	return topo
}

func (hwInfo defaultHwInfoSource) getCpusetFilter() func(lp int) bool {
	cpusetNameB, e := os.ReadFile("/proc/self/cpuset")
	if e != nil {
		logger.Warn("cannot determine cpuset name",
			zap.Error(e),
		)
		return func(int) bool { return true }
	}

	cpusetName := strings.Trim(string(cpusetNameB), "/\n")
	filenames := []string{
		path.Join("/sys/fs/cgroup/cpuset", cpusetName, "cpuset.effective_cpus"), // systemd, cgroup v1
		path.Join("/sys/fs/cgroup", cpusetName, "cpuset.cpus.effective"),        // systemd, cgroup v2
		"/sys/fs/cgroup/cpuset/cpuset.effective_cpus",                           // Docker, cgroup v1
		"/sys/fs/cgroup/cpuset.cpus.effective",                                  // Docker, cgroup v1
	}

	var cpuList []int
	for _, filename := range filenames {
		b, e := os.ReadFile(filename)
		if e != nil {
			continue
		}

		cpuList, e = cpuset.Parse(string(b))
		if e != nil {
			break
		}
		logger.Debug("retrieved cpuset",
			zap.String("filename", filename),
			zap.Ints("list", cpuList),
		)
	}

	if cpuList == nil {
		logger.Warn("cannot retrieve cpuset",
			zap.Strings("filenames", filenames),
		)
		return func(int) bool { return true }
	}

	m := make(map[int]bool)
	for _, cpu := range cpuList {
		m[cpu] = true
	}
	return func(lp int) bool { return m[lp] }
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
