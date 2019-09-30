package fibmgmt

import (
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type FibInfo struct {
	NEntries int // Number of entries.
}

type NameArg struct {
	Name *ndn.Name
}

type InsertArg struct {
	Name       *ndn.Name
	Nexthops   []iface.FaceId
	StrategyId int
}

type InsertReply struct {
	IsNew bool
}

type LookupReply struct {
	HasEntry   bool
	Name       *ndn.Name
	Nexthops   []iface.FaceId
	StrategyId int
}
