package strategymgmt

import (
	"errors"

	"ndn-dpdk/container/strategycode"
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
	sc, ok := strategycode.Get(args.Id)
	if !ok {
		return errors.New("strategy not found")
	}
	*reply = makeStrategyInfo(sc)
	return nil
}

func (StrategyMgmt) Load(args LoadArg, reply *StrategyInfo) error {
	if _, ok := strategycode.Find(args.Name); ok {
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
	sc, ok := strategycode.Get(args.Id)
	if !ok {
		return errors.New("strategy not found")
	}
	*reply = makeStrategyInfo(sc)
	return sc.Close()
}
