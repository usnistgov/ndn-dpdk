package ndtmgmt

import (
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/ndn"
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

func (mg NdtMgmt) Update(args UpdateArgs, reply *UpdateReply) error {
	if args.Name != "" {
		name, e := ndn.ParseName(args.Name)
		if e != nil {
			return e
		}
		args.Hash = mg.Ndt.ComputeHash(name)
	}
	reply.Index = mg.Ndt.Update(args.Hash, args.Value)
	return nil
}
