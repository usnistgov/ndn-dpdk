package strategymgmt

import (
	"ndn-dpdk/container/strategycode"
)

type IdArg struct {
	Id int
}

type StrategyInfo struct {
	Id   int
	Name string
}

func makeStrategyInfo(sc strategycode.StrategyCode) (si StrategyInfo) {
	si.Id = sc.GetId()
	si.Name = sc.GetName()
	return si
}

type LoadArg struct {
	Name string
	Elf  []byte
}
