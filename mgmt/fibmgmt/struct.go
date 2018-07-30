package fibmgmt

import (
	"ndn-dpdk/iface"
)

type FibInfo struct {
	NEntries    int // Number of entries, counting duplicates only once.
	NEntriesDup int // Number of entries, counting duplicates multiple times.
	NVirtuals   int // Number of virtual entries for two-stage LPM.
	NNodes      int // Number of tree nodes.
}

type NameArg struct {
	Name string
}

type InsertArg struct {
	Name     string
	Nexthops []iface.FaceId
}

type InsertReply struct {
	IsNew bool
}

type LookupReply struct {
	HasEntry   bool
	Name       string
	Nexthops   []iface.FaceId
	StrategyId int
}
