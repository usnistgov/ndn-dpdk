package strategymgmt

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/container/strategycode"
)

type StrategyMgmt struct{}

func (StrategyMgmt) List(args struct{}, reply *[]StrategyInfo) error {
	scl := strategycode.List()
	*reply = make([]StrategyInfo, len(scl))
	for i, sc := range scl {
		(*reply)[i] = makeStrategyInfo(sc)
	}
	return nil
}

func (StrategyMgmt) Get(args IdArg, reply *StrategyInfo) error {
	sc := strategycode.Get(args.Id)
	if sc == nil {
		return errors.New("strategy not found")
	}
	*reply = makeStrategyInfo(sc)
	return nil
}

func (StrategyMgmt) Load(args LoadArg, reply *StrategyInfo) error {
	if strategycode.Find(args.Name) != nil {
		return errors.New("duplicate name")
	}

	sc, e := strategycode.Load(args.Name, args.Elf)
	if e != nil {
		return e
	}

	*reply = makeStrategyInfo(sc)
	return nil
}

func (StrategyMgmt) Unload(args IdArg, reply *StrategyInfo) error {
	sc := strategycode.Get(args.Id)
	if sc == nil {
		return errors.New("strategy not found")
	}
	*reply = makeStrategyInfo(sc)
	return sc.Close()
}

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
