package disk

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// StoreCounters contains disk store counters.
type StoreCounters struct {
	NPutDataBegin   uint64 `json:"nPutDataBegin"`
	NPutDataSuccess uint64 `json:"nPutDataSuccess"`
	NPutDataFailure uint64 `json:"nPutDataFailure"`
	NGetDataBegin   uint64 `json:"nGetDataBegin"`
	NGetDataReuse   uint64 `json:"nGetDataReuse"`
	NGetDataSuccess uint64 `json:"nGetDataSuccess"`
	NGetDataFailure uint64 `json:"nGetDataFailure"`
}

// Counters retrieves disk store counters.
func (store *Store) Counters() (cnt StoreCounters) {
	cnt.NPutDataBegin = uint64(store.c.nPutDataBegin)
	cnt.NPutDataSuccess = uint64(store.c.nPutDataFinish[1])
	cnt.NPutDataFailure = uint64(store.c.nPutDataFinish[0])
	cnt.NGetDataBegin = uint64(store.c.nGetDataBegin)
	cnt.NGetDataReuse = uint64(store.c.nGetDataReuse)
	cnt.NGetDataSuccess = uint64(store.c.nGetDataSuccess)
	cnt.NGetDataFailure = uint64(store.c.nGetDataFailure)
	return cnt
}

// GqlStoreCountersType is the GraphQL type for StoreCounters.
var GqlStoreCountersType = graphql.NewObject(graphql.ObjectConfig{
	Name:   "DiskStoreCounters",
	Fields: gqlserver.BindFields(StoreCounters{}, nil),
})
