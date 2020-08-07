// Package fibtestenv provides utilities for FIB unit tests.
package fibtestenv

import (
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

var dummyStrategy strategycode.StrategyCode

// DummyStrategy returns an empty strategy.
func DummyStrategy() strategycode.StrategyCode {
	if dummyStrategy == nil {
		dummyStrategy = strategycode.MakeEmpty("fibtestenv.dummyStrategy")
	}
	return dummyStrategy
}

// MakeEntry builds a fibdef.Entry.
//  name: ndn.Name or string
//  sc: int or strategycode.StrategyCode or nil
func MakeEntry(name, sc interface{}, nexthops ...iface.ID) (entry fibdef.Entry) {
	switch n := name.(type) {
	case ndn.Name:
		entry.Name = n
	case string:
		entry.Name = ndn.ParseName(n)
	default:
		panic(name)
	}

	switch s := sc.(type) {
	case int:
		entry.Strategy = s
	case strategycode.StrategyCode:
		entry.Strategy = s.GetId()
	case nil:
		entry.Strategy = DummyStrategy().GetId()
	default:
		panic(sc)
	}

	entry.Nexthops = nexthops
	return entry
}
