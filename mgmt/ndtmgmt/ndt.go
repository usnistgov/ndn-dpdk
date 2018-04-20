package ndtmgmt

import (
	"ndn-dpdk/container/ndt"
)

type NdtMgmt struct {
	Ndt ndt.Ndt
}

func (mg NdtMgmt) ReadTable(args struct{}, reply *[]uint8) error {
	*reply = mg.Ndt.ReadTable()
	return nil
}

func (mg NdtMgmt) ReadCounters(args struct{}, reply *[]int) error {
	*reply = mg.Ndt.ReadCounters()
	return nil
}

func (mg NdtMgmt) Update(args UpdateArgs, reply *struct{}) error {
	for _, instn := range args.Instructions {
		mg.Ndt.Update(instn.Hash, instn.Value)
	}
	return nil
}
