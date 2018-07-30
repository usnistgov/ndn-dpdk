package fibmgmt

import (
	"ndn-dpdk/container/fib"
	"ndn-dpdk/ndn"
)

// Strategy for new FIB entries.
var TheStrategy fib.StrategyCode

type FibMgmt struct {
	Fib *fib.Fib
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

	name, e := ndn.ParseName(args.Name)
	if e != nil {
		return e
	}
	entry.SetName(name)

	e = entry.SetNexthops(args.Nexthops)
	if e != nil {
		return e
	}

	entry.SetStrategy(TheStrategy)

	isNew, e := mg.Fib.Insert(entry)
	if e != nil {
		return e
	}

	reply.IsNew = isNew
	return nil
}

func (mg FibMgmt) Erase(args NameArg, reply *struct{}) error {
	name, e := ndn.ParseName(args.Name)
	if e != nil {
		return e
	}

	return mg.Fib.Erase(name)
}

func (mg FibMgmt) Find(args NameArg, reply *LookupReply) error {
	return mg.lookup(args, reply, mg.Fib.Find)
}

func (mg FibMgmt) Lpm(args NameArg, reply *LookupReply) error {
	return mg.lookup(args, reply, mg.Fib.Lpm)
}

func (mg FibMgmt) lookup(args NameArg, reply *LookupReply, lookup func(name *ndn.Name) *fib.Entry) error {
	name, e := ndn.ParseName(args.Name)
	if e != nil {
		return e
	}

	entry := lookup(name)
	if entry != nil {
		reply.HasEntry = true
		reply.Name = entry.GetName().String()
		reply.Nexthops = entry.GetNexthops()
		reply.StrategyId = entry.GetStrategy().GetId()
	}
	return nil
}

func (mg FibMgmt) ReadEntryCounters(args NameArg, reply *fib.EntryCounters) error {
	name, e := ndn.ParseName(args.Name)
	if e != nil {
		return e
	}

	*reply = mg.Fib.ReadEntryCounters(name)
	return nil
}
