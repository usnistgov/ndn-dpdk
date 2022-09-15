package hwinfo

import (
	"fmt"
	"math/big"

	procinfo "github.com/c9s/goprocinfo/linux"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

const (
	pathCPUInfo       = "/proc/cpuinfo"
	pathProcessStatus = "/proc/self/status"
	pathSystemNode    = "/sys/devices/system/node"
	maxPhysicalCore   = 4096
	maxNumaNode       = 32
)

type procinfoProvider struct {
	cachedCores Cores
}

func (p *procinfoProvider) Cores() (cores Cores) {
	if len(p.cachedCores) > 0 {
		return p.cachedCores
	}

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

	for _, processor := range cpuInfo.Processors {
		if allowed.Bit(int(processor.Id)) == 0 || processor.CoreId >= maxPhysicalCore {
			continue
		}
		numa, ok := p.findNumaSocket(processor)
		if !ok {
			continue
		}
		cores = append(cores, CoreInfo{
			ID:          int(processor.Id),
			NumaSocket:  numa,
			PhysicalKey: maxPhysicalCore*int(processor.PhysicalId) + int(processor.CoreId),
		})
	}

	p.cachedCores = cores
	return cores
}

func (procinfoProvider) findNumaSocket(processor procinfo.Processor) (int, bool) {
	for i := 0; i < maxNumaNode; i++ {
		path := fmt.Sprintf("%s/node%d/cpu%d", pathSystemNode, i, processor.Id)
		if unix.Access(path, unix.F_OK) == nil {
			return i, true
		}
	}
	return -1, false
}
