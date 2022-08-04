// Package fibtestenv provides utilities for FIB unit tests.
package fibtestenv

import (
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

var dummyStrategy *strategycode.Strategy

// DummyStrategy returns an empty strategy.
func DummyStrategy() *strategycode.Strategy {
	if dummyStrategy == nil {
		dummyStrategy = strategycode.MakeEmpty("fibtestenv.dummyStrategy")
	}
	return dummyStrategy
}

// MakeEntry builds a fibdef.Entry.
//
//	name: ndn.Name or string
//	sc: int or *strategycode.Strategy or nil
func MakeEntry(name, sc any, nexthops ...iface.ID) (entry fibdef.Entry) {
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
	case *strategycode.Strategy:
		entry.Strategy = s.ID()
	case nil:
		entry.Strategy = DummyStrategy().ID()
	default:
		panic(sc)
	}

	entry.Nexthops = nexthops
	return entry
}
