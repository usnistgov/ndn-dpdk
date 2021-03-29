package ealconfig

import (
	"math/big"

	procinfo "github.com/c9s/goprocinfo/linux"
	"github.com/pkg/math"
	"go.uber.org/zap"
)

const (
	pathCPUInfo       = "/proc/cpuinfo"
	pathProcessStatus = "/proc/self/status"
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
	status, e := procinfo.ReadProcessStatus(pathProcessStatus)
	if e != nil {
		logger.Panic(pathProcessStatus, zap.Error(e))
	}
	allowed := &big.Int{}
	for _, word := range status.CpusAllowed {
		allowed.Lsh(allowed, 32)
		allowed.Add(allowed, big.NewInt(int64(word)))
	}

	cpuInfo, e := procinfo.ReadCPUInfo(pathCPUInfo)
	if e != nil {
		logger.Panic(pathCPUInfo, zap.Error(e))
	}

	hasHyperThread := make(map[[2]int64]bool)
	for _, processor := range cpuInfo.Processors {
		if allowed.Bit(int(processor.Id)) == 0 {
			continue
		}
		htKey := [2]int64{processor.PhysicalId, processor.CoreId}
		list = append(list, CoreInfo{
			ID:          int(processor.Id),
			NumaSocket:  int(processor.PhysicalId),
			HyperThread: hasHyperThread[htKey],
		})
		hasHyperThread[htKey] = true
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
