package hwinfo

import (
	"math/big"

	procinfo "github.com/c9s/goprocinfo/linux"
	"go.uber.org/zap"
)

const (
	pathCPUInfo       = "/proc/cpuinfo"
	pathProcessStatus = "/proc/self/status"
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
		if allowed.Bit(int(processor.Id)) == 0 {
			continue
		}
		cores = append(cores, CoreInfo{
			NumaSocket:   int(processor.PhysicalId),
			PhysicalCore: int(processor.CoreId),
			LogicalCore:  int(processor.Id),
		})
	}

	p.cachedCores = cores
	return cores
}
