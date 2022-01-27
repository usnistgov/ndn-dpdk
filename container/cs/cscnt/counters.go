// Package cscnt provides Content Store counters.
package cscnt

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// Counters contains CS counters.
type Counters struct {
	NHits            uint64 `json:"nHits" gqldesc:"Lookup hits."`
	NMisses          uint64 `json:"nMisses" gqldesc:"Lookup misses."`
	DirectEntries    int    `json:"directEntries" gqldesc:"Direct entries."`
	DirectCapacity   int    `json:"directCapacity" gqldesc:"Direct capacity."`
	IndirectEntries  int    `json:"indirectEntries" gqldesc:"Indirect entries."`
	IndirectCapacity int    `json:"indirectCapacity" gqldesc:"Indirect capacity."`
}

// ReadCounters retrieves CS counters from PIT and CS.
func ReadCounters(p *pit.Pit, c *cs.Cs) (cnt Counters) {
	pitCnt := p.Counters()
	cnt.NHits = pitCnt.NCsMatch
	cnt.NMisses = pitCnt.NInsert + pitCnt.NFound
	cnt.DirectEntries, cnt.DirectCapacity = readCslCnt(c, cs.ListDirect)
	cnt.IndirectEntries, cnt.IndirectCapacity = readCslCnt(c, cs.ListIndirect)
	return cnt
}

func readCslCnt(c *cs.Cs, list cs.ListID) (nEntries, capacity int) {
	return c.CountEntries(list), c.Capacity(list)
}

// GqlCountersType is the GraphQL type for Counters.
var GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
	Name:   "CsCounters",
	Fields: gqlserver.BindFields(Counters{}, nil),
})
