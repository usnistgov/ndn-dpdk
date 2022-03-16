package fwdp

import (
	"errors"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var (
	// GqlDataPlane is the DataPlane instance accessible via GraphQL.
	GqlDataPlane *DataPlane

	errNoGqlDataPlane = errors.New("DataPlane unavailable")
)

// GraphQL types.
var (
	GqlInputNodeType   *gqlserver.NodeType
	GqlInputType       *graphql.Object
	GqlFwdNodeType     *gqlserver.NodeType
	GqlFwdType         *graphql.Object
	GqlDataPlaneType   *graphql.Object
	GqlFwdCountersType *graphql.Object
)

func init() {
	GqlInputNodeType = gqlserver.NewNodeType((*Input)(nil))
	GqlInputNodeType.Retrieve = func(id string) (interface{}, error) {
		if GqlDataPlane == nil {
			return nil, errNoGqlDataPlane
		}
		i, e := strconv.Atoi(id)
		if e != nil || i < 0 || i >= len(GqlDataPlane.fwis) {
			return nil, nil
		}
		return GqlDataPlane.fwis[i], nil
	}

	GqlInputType = graphql.NewObject(GqlInputNodeType.Annotate(graphql.ObjectConfig{
		Name: "FwInput",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Input thread index.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					input := p.Source.(*Input)
					return input.id, nil
				},
			},
			"worker": ealthread.GqlWithWorker(nil),
		},
	}))
	GqlInputNodeType.Register(GqlInputType)

	GqlFwdNodeType = gqlserver.NewNodeType((*Fwd)(nil))
	GqlFwdNodeType.Retrieve = func(id string) (interface{}, error) {
		if GqlDataPlane == nil {
			return nil, errNoGqlDataPlane
		}
		i, e := strconv.Atoi(id)
		if e != nil || i < 0 || i >= len(GqlDataPlane.fwds) {
			return nil, nil
		}
		return GqlDataPlane.fwds[i], nil
	}

	GqlFwdType = graphql.NewObject(GqlFwdNodeType.Annotate(graphql.ObjectConfig{
		Name: "FwFwd",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Forwarding thread index.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					return fwd.id, nil
				},
			},
			"worker": ealthread.GqlWithWorker(nil),
		},
	}))
	GqlFwdNodeType.Register(GqlFwdType)

	GqlDataPlaneType = graphql.NewObject(graphql.ObjectConfig{
		Name: "FwDataPlane",
		Fields: graphql.Fields{
			"inputs": &graphql.Field{
				Description: "Input threads.",
				Type:        gqlserver.NewNonNullList(GqlInputType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					dp := p.Source.(*DataPlane)
					return dp.fwis, nil
				},
			},
			"fwds": &graphql.Field{
				Description: "Forwarding threads.",
				Type:        gqlserver.NewNonNullList(GqlFwdType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					dp := p.Source.(*DataPlane)
					return dp.fwds, nil
				},
			},
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "fwdp",
		Description: "Forwarder data plane.",
		Type:        GqlDataPlaneType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return GqlDataPlane, nil
		},
	})

	GqlFwdCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FwFwdCounters",
		Fields: gqlserver.BindFields(FwdCounters{}, nil),
	})
	GqlFwdCountersType.AddFieldConfig("inputLatency", &graphql.Field{
		Description: "Latency between packet arrival and dequeuing at forwarding thread, in nanoseconds.",
		Type:        graphql.NewNonNull(runningstat.GqlSnapshotType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			index := p.Source.(FwdCounters).id
			fwd := GqlDataPlane.fwds[index]
			latencyStat := runningstat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
			return latencyStat.Read().Scale(eal.TscNanos), nil
		},
	})
	for t, plural := range map[ndni.PktType]string{ndni.PktInterest: "Interests", ndni.PktData: "Data", ndni.PktNack: "Nacks"} {
		t := t
		GqlFwdCountersType.AddFieldConfig(fmt.Sprintf("n%sQueued", plural), &graphql.Field{
			Description: fmt.Sprintf("%s queued in input thread.", plural),
			Type:        gqlserver.NonNullUint64,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				index := p.Source.(FwdCounters).id
				var sum uint64
				for _, input := range GqlDataPlane.fwis {
					sum += input.rxl.DemuxOf(t).DestCounters(index).NQueued
				}
				return sum, nil
			},
		})
		GqlFwdCountersType.AddFieldConfig(fmt.Sprintf("n%sDropped", plural), &graphql.Field{
			Description: fmt.Sprintf("%s dropped in input thread.", plural),
			Type:        gqlserver.NonNullUint64,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				index := p.Source.(FwdCounters).id
				var sum uint64
				for _, input := range GqlDataPlane.fwis {
					sum += input.rxl.DemuxOf(t).DestCounters(index).NDropped
				}
				return sum, nil
			},
		})
		GqlFwdCountersType.AddFieldConfig(fmt.Sprintf("n%sCongMarked", plural), &graphql.Field{
			Description: fmt.Sprintf("Congestion marks added to %s.", plural),
			Type:        gqlserver.NonNullUint64,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				index := p.Source.(FwdCounters).id
				fwd := GqlDataPlane.fwds[index]
				q := fwd.PktQueueOf(t)
				return q.Counters().NDrops, nil
			},
		})
	}

	fwdCountersConfigTemplate := gqlserver.CountersConfig{
		FindArgs: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "Forwarding thread ID.",
				Type:        gqlserver.NonNullID,
			},
		},
		Find: func(p graphql.ResolveParams) (source interface{}, enders []interface{}, e error) {
			id := p.Args["id"].(string)
			var fwd *Fwd
			if e := gqlserver.RetrieveNodeOfType(GqlFwdNodeType, id, &fwd); e != nil {
				return nil, nil, e
			}
			return fwd, nil, nil
		},
	}
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "Forwarding thread counters in forwarder data plane.",
		Parent:       GqlFwdType,
		Name:         "counters",
		Subscription: "fwFwdCounters",
		NoDiff:       true,
		FindArgs:     fwdCountersConfigTemplate.FindArgs,
		Find:         fwdCountersConfigTemplate.Find,
		Type:         graphql.NewNonNull(GqlFwdCountersType),
		Read: func(p graphql.ResolveParams) (interface{}, error) {
			fwd := p.Source.(*Fwd)
			return fwd.Counters(), nil
		},
	})
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "PIT counters in forwarder data plane.",
		Parent:       GqlFwdType,
		Name:         "pitCounters",
		Subscription: "fwPitCounters",
		FindArgs:     fwdCountersConfigTemplate.FindArgs,
		Find:         fwdCountersConfigTemplate.Find,
		Type:         graphql.NewNonNull(pit.GqlCountersType),
		Read: func(p graphql.ResolveParams) (interface{}, error) {
			fwd := p.Source.(*Fwd)
			return fwd.Pit().Counters(), nil
		},
	})
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "CS counters in forwarder data plane.",
		Parent:       GqlFwdType,
		Name:         "csCounters",
		Subscription: "fwCsCounters",
		FindArgs:     fwdCountersConfigTemplate.FindArgs,
		Find:         fwdCountersConfigTemplate.Find,
		Type:         graphql.NewNonNull(cs.GqlCountersType),
		Read: func(p graphql.ResolveParams) (interface{}, error) {
			fwd := p.Source.(*Fwd)
			return fwd.Cs().Counters(), nil
		},
	})

	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "DiskStore counters in forwarder data plane.",
		Parent:       GqlDataPlaneType,
		Name:         "diskCounters",
		Subscription: "fwDiskCounters",
		Find: func(p graphql.ResolveParams) (source interface{}, enders []interface{}, e error) {
			return GqlDataPlane, nil, nil
		},
		Type: disk.GqlStoreCountersType,
		Read: func(p graphql.ResolveParams) (interface{}, error) {
			dp := p.Source.(*DataPlane)
			if dp.fwdisk == nil {
				return nil, nil
			}
			return dp.fwdisk.store.Counters(), nil
		},
	})
}
