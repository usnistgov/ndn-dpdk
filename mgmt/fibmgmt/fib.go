package fibmgmt

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
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
	for _, entry := range mg.Fib.List() {
		*reply = append(*reply, entry.Name.String())
	}
	return nil
}

func (mg FibMgmt) Insert(args InsertArg, reply *struct{}) error {
	entry := fibdef.Entry{
		Name: args.Name,
	}
	entry.Nexthops = args.Nexthops

	strategyId := args.StrategyId
	if strategyId == 0 {
		strategyId = mg.DefaultStrategyId
	}
	if sc := strategycode.Get(strategyId); sc != nil {
		entry.Strategy = sc.GetId()
	} else {
		return errors.New("strategy not found")
	}

	e := mg.Fib.Insert(entry)
	if e != nil {
		return e
	}

	return nil
}

func (mg FibMgmt) Erase(args NameArg, reply *struct{}) error {
	return mg.Fib.Erase(args.Name)
}

func (mg FibMgmt) Find(args NameArg, reply *LookupReply) error {
	entry := mg.Fib.Find(args.Name)
	if entry != nil {
		reply.HasEntry = true
		reply.Name = entry.Name
		reply.Nexthops = entry.Nexthops
		reply.StrategyId = entry.Strategy
	}
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

type LookupReply struct {
	HasEntry   bool
	Name       ndn.Name
	Nexthops   []iface.ID
	StrategyId int
}
