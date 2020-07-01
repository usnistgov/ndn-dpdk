package fibmgmt

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

type FibMgmt struct {
	Fib               *fib.Fib
	DefaultStrategyId int
}

func (mg FibMgmt) Info(args struct{}, reply *FibInfo) error {
	reply.NEntries = mg.Fib.Len()
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
	if sc := strategycode.Get(strategyId); sc != nil {
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

func (mg FibMgmt) lookup(args NameArg, reply *LookupReply, lookup func(name ndn.Name) *fib.Entry) error {
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

type FibInfo struct {
	NEntries int // Number of entries.
}

type NameArg struct {
	Name ndn.Name
}

type InsertArg struct {
	Name       ndn.Name
	Nexthops   []iface.ID
	StrategyId int
}

type InsertReply struct {
	IsNew bool
}

type LookupReply struct {
	HasEntry   bool
	Name       ndn.Name
	Nexthops   []iface.ID
	StrategyId int
}
