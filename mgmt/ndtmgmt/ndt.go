package ndtmgmt

import (
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/container/ndt/ndtupdater"
	"ndn-dpdk/ndn"
)

type NdtMgmt struct {
	Ndt     *ndt.Ndt
	Updater *ndtupdater.NdtUpdater
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
	if args.Name != nil {
		args.Hash = mg.Ndt.ComputeHash(args.Name)
	}
	reply.Index = mg.Ndt.GetIndex(args.Hash)
	mg.Updater.Update(reply.Index, args.Value)
	return nil
}

type UpdateArgs struct {
	Hash  uint64
	Name  *ndn.Name // If not empty, overrides Hash with the hash of this name.
	Value uint8
}

type UpdateReply struct {
	Index uint64
}
