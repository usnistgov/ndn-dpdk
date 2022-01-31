package cs

/*
#include "../../csrc/pcct/cs.h"
*/
import "C"
import (
	"unsafe"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/pcct"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// ReadMissCounter reads CS miss counters.
// This is assigned during package pit initialization.
var ReadMissCounter func(pcct *pcct.Pcct) uint64

// Counters contains CS counters.
type Counters struct {
	DirectEntries    int `json:"directEntries" gqldesc:"Direct entries."`
	DirectCapacity   int `json:"directCapacity" gqldesc:"Direct capacity."`
	IndirectEntries  int `json:"indirectEntries" gqldesc:"Indirect entries."`
	IndirectCapacity int `json:"indirectCapacity" gqldesc:"Indirect capacity."`

	NHitMemory   uint64 `json:"nHitMemory" gqldesc:"Lookup hits on memory entry."`
	NHitDisk     uint64 `json:"nHitDisk" gqldesc:"Lookup hits on disk entry."`
	NHitIndirect uint64 `json:"nHitIndirect" gqldesc:"Lookup hits on indirect entry."`
	NMiss        uint64 `json:"nMiss" gqldesc:"Lookup misses."`
	NDiskInsert  uint64 `json:"nDiskInsert" gqldesc:"Packets written to disk."`
	NDiskDelete  uint64 `json:"nDiskDelete" gqldesc:"Packets deleted from disk."`
	NDiskFull    uint64 `json:"nDiskFull" gqldesc:"Packets not written to disk due to allocation error."`
}

// Counters retrieves CS counters.
func (cs *Cs) Counters() (cnt Counters) {
	cnt.DirectEntries = cs.CountEntries(ListDirect)
	cnt.DirectCapacity = cs.Capacity(ListDirect)
	cnt.IndirectEntries = cs.CountEntries(ListIndirect)
	cnt.IndirectCapacity = cs.Capacity(ListIndirect)

	cnt.NHitMemory = uint64(cs.nHitMemory)
	cnt.NHitDisk = uint64(cs.nHitDisk)
	cnt.NHitIndirect = uint64(cs.nHitIndirect)
	cnt.NMiss = ReadMissCounter((*pcct.Pcct)(unsafe.Pointer(C.Pcct_FromCs(cs.ptr()))))
	cnt.NDiskInsert = uint64(cs.nDiskInsert)
	cnt.NDiskDelete = uint64(cs.nDiskDelete)
	cnt.NDiskFull = uint64(cs.nDiskFull)
	return cnt
}

// GqlCountersType is the GraphQL type for Counters.
var GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
	Name:   "CsCounters",
	Fields: gqlserver.BindFields(Counters{}, nil),
})
