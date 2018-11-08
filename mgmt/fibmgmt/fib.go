package fibmgmt

import (
	"errors"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/container/strategycode"
	"ndn-dpdk/ndn"
)

type FibMgmt struct {
	Fib               *fib.Fib
	DefaultStrategyId int
}

func (mg FibMgmt) Info(args struct{}, reply *FibInfo) error {
	reply.NEntries = mg.Fib.CountEntries(false)
	reply.NEntriesDup = mg.Fib.CountEntries(true)
	reply.NVirtuals = mg.Fib.CountVirtuals()
	reply.NNodes = mg.Fib.CountNodes()
	return nil
}

func (mg FibMgmt) List(args struct{}, reply *[]string) error {
	*reply = make([]string, 0)
	for _, name := range mg.Fib.ListNames() {
		*reply = append(*reply, name.String())
	}
	return nil
}

func (mg FibMgmt) Insert(args InsertArg, reply *InsertReply) error {
	entry := new(fib.Entry)

	entry.SetName(args.Name)

	if e := entry.SetNexthops(args.Nexthops); e != nil {
		return e
	}

	strategyId := args.StrategyId
	if strategyId == 0 {
		strategyId = mg.DefaultStrategyId
	}
	if sc, ok := strategycode.Get(strategyId); ok {
		entry.SetStrategy(sc)
	} else {
		return errors.New("strategy not found")
	}

	isNew, e := mg.Fib.Insert(entry)
	if e != nil {
		return e
	}

	reply.IsNew = isNew
	return nil
}

func (mg FibMgmt) Erase(args NameArg, reply *struct{}) error {
	return mg.Fib.Erase(args.Name)
}

func (mg FibMgmt) Find(args NameArg, reply *LookupReply) error {
	return mg.lookup(args, reply, mg.Fib.Find)
}

func (mg FibMgmt) Lpm(args NameArg, reply *LookupReply) error {
	return mg.lookup(args, reply, mg.Fib.Lpm)
}

func (mg FibMgmt) lookup(args NameArg, reply *LookupReply, lookup func(name *ndn.Name) *fib.Entry) error {
	entry := lookup(args.Name)
	if entry != nil {
		reply.HasEntry = true
		reply.Name = entry.GetName()
		reply.Nexthops = entry.GetNexthops()
		reply.StrategyId = entry.GetStrategy().GetId()
	}
	return nil
}

func (mg FibMgmt) ReadEntryCounters(args NameArg, reply *fib.EntryCounters) error {
	*reply = mg.Fib.ReadEntryCounters(args.Name)
	return nil
}
