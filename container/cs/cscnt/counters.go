// Package cscnt provides Content Store counters.
package cscnt

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/pit"
)

// Counters contains CS counters.
type Counters struct {
	NHits            uint64 `json:"nHits"`            // lookup hits
	NMisses          uint64 `json:"nMisses"`          // lookup misses
	DirectEntries    int    `json:"directEntries"`    // direct entries
	DirectCapacity   int    `json:"directCapacity"`   // direct capacity
	IndirectEntries  int    `json:"indirectEntries"`  // indirect entries
	IndirectCapacity int    `json:"indirectCapacity"` // indirect capacity
}

// ReadCounters retrieves CS counters from PIT and CS.
func ReadCounters(p *pit.Pit, c *cs.Cs) (cnt Counters) {
	pitCnt := p.Counters()
	cnt.NHits = pitCnt.NCsMatch
	cnt.NMisses = pitCnt.NInsert + pitCnt.NFound
	cnt.DirectEntries, cnt.DirectCapacity = readCslCnt(c, cs.ListMd)
	cnt.IndirectEntries, cnt.IndirectCapacity = readCslCnt(c, cs.ListMi)
	return cnt
}

func readCslCnt(c *cs.Cs, list cs.ListID) (nEntries, capacity int) {
	return c.CountEntries(list), c.Capacity(list)
}

// GqlCountersType is the GraphQL type for Counters.
var GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
	Name:   "CsCounters",
	Fields: graphql.BindFields(Counters{}),
})
