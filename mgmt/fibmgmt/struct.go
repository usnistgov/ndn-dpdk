package fibmgmt

import (
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type FibInfo struct {
	NEntries    int // Number of entries, counting duplicates only once.
	NEntriesDup int // Number of entries, counting duplicates multiple times.
	NVirtuals   int // Number of virtual entries for two-stage LPM.
	NNodes      int // Number of tree nodes.
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
