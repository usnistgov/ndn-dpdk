package fwdp

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/rttest"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var (
	// GqlDataPlane is the DataPlane instance accessible via GraphQL.
	GqlDataPlane *DataPlane

	errNoGqlDataPlane = errors.New("DataPlane unavailable")
)

type gqlFibNexthopRttRecord struct {
	*rttest.RttEstimator
	Face iface.Face `json:"face"`
	Fwd  *Fwd       `json:"fwd"`
}

// GraphQL types.
var (
	GqlInputType         *gqlserver.NodeType[*Input]
	GqlFwdType           *gqlserver.NodeType[*Fwd]
	GqlDataPlaneType     *graphql.Object
	GqlFwdCountersType   *graphql.Object
	GqlFibNexthopRttType *graphql.Object
)

func init() {
	GqlInputType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "FwInput",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Input thread index.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					input := p.Source.(*Input)
					return input.id, nil
				},
			},
			"worker": ealthread.GqlWithWorker(nil),
		},
	}, gqlserver.NodeConfig[*Input]{
		RetrieveInt: func(id int) *Input {
			if GqlDataPlane == nil {
				return nil
			}
			if id < 0 || id >= len(GqlDataPlane.fwis) {
				return nil
			}
			return GqlDataPlane.fwis[id]
		},
	})

	GqlFwdType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "FwFwd",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Forwarding thread index.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					fwd := p.Source.(*Fwd)
					return fwd.id, nil
				},
			},
			"worker": ealthread.GqlWithWorker(nil),
		},
	}, gqlserver.NodeConfig[*Fwd]{
		RetrieveInt: func(id int) *Fwd {
			if GqlDataPlane == nil {
				return nil
			}
			if id < 0 || id >= len(GqlDataPlane.fwds) {
				return nil
			}
			return GqlDataPlane.fwds[id]
		},
	})

	GqlDataPlaneType = graphql.NewObject(graphql.ObjectConfig{
		Name: "FwDataPlane",
		Fields: graphql.Fields{
			"inputs": &graphql.Field{
				Description: "Input threads.",
				Type:        gqlserver.NewNonNullList(GqlInputType.Object),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					dp := p.Source.(*DataPlane)
					return dp.fwis, nil
				},
			},
			"fwds": &graphql.Field{
				Description: "Forwarding threads.",
				Type:        gqlserver.NewNonNullList(GqlFwdType.Object),
				Resolve: func(p graphql.ResolveParams) (any, error) {
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
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return GqlDataPlane, nil
		},
	})

	GqlFwdCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FwFwdCounters",
		Fields: gqlserver.BindFields[FwdCounters](nil),
	})
	GqlFwdCountersType.AddFieldConfig("inputLatency", &graphql.Field{
		Description: "Latency between packet arrival and dequeuing at forwarding thread, in nanoseconds.",
		Type:        graphql.NewNonNull(runningstat.GqlSnapshotType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			index := p.Source.(FwdCounters).id
			fwd := GqlDataPlane.fwds[index]
			return fwd.LatencyStat().Read().Scale(eal.TscNanos), nil
		},
	})
	for t, plural := range map[ndni.PktType]string{ndni.PktInterest: "Interests", ndni.PktData: "Data", ndni.PktNack: "Nacks"} {
		t := t
		GqlFwdCountersType.AddFieldConfig(fmt.Sprintf("n%sQueued", plural), &graphql.Field{
			Description: fmt.Sprintf("%s queued in input thread.", plural),
			Type:        gqlserver.NonNullUint64,
			Resolve: func(p graphql.ResolveParams) (any, error) {
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
			Resolve: func(p graphql.ResolveParams) (any, error) {
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
			Resolve: func(p graphql.ResolveParams) (any, error) {
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
		Find: func(p graphql.ResolveParams) (source any, enders []any, e error) {
			return GqlFwdType.Retrieve(p.Args["id"].(string)), nil, nil
		},
	}
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "Forwarding thread counters in forwarder data plane.",
		Parent:       GqlFwdType.Object,
		Name:         "counters",
		Subscription: "fwFwdCounters",
		NoDiff:       true,
		FindArgs:     fwdCountersConfigTemplate.FindArgs,
		Find:         fwdCountersConfigTemplate.Find,
		Type:         graphql.NewNonNull(GqlFwdCountersType),
		Read: func(p graphql.ResolveParams) (any, error) {
			fwd := p.Source.(*Fwd)
			return fwd.Counters(), nil
		},
	})
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "PIT counters in forwarder data plane.",
		Parent:       GqlFwdType.Object,
		Name:         "pitCounters",
		Subscription: "fwPitCounters",
		FindArgs:     fwdCountersConfigTemplate.FindArgs,
		Find:         fwdCountersConfigTemplate.Find,
		Type:         graphql.NewNonNull(pit.GqlCountersType),
		Read: func(p graphql.ResolveParams) (any, error) {
			fwd := p.Source.(*Fwd)
			return fwd.Pit().Counters(), nil
		},
	})
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "CS counters in forwarder data plane.",
		Parent:       GqlFwdType.Object,
		Name:         "csCounters",
		Subscription: "fwCsCounters",
		FindArgs:     fwdCountersConfigTemplate.FindArgs,
		Find:         fwdCountersConfigTemplate.Find,
		Type:         graphql.NewNonNull(cs.GqlCountersType),
		Read: func(p graphql.ResolveParams) (any, error) {
			fwd := p.Source.(*Fwd)
			return fwd.Cs().Counters(), nil
		},
	})

	GqlFibNexthopRttType = graphql.NewObject(graphql.ObjectConfig{
		Name:        "FibNexthopRtt",
		Description: "FIB nexthop and RTT measurements in a forwarding thread.",
		Fields: graphql.Fields{
			"face": &graphql.Field{Type: iface.GqlFaceType.Object},
			"fwd":  &graphql.Field{Type: graphql.NewNonNull(GqlFwdType.Object)},
			"srtt": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Float),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(gqlFibNexthopRttRecord).SRTT().Seconds(), nil
				},
			},
			"rttvar": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Float),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(gqlFibNexthopRttRecord).RTTVAR().Seconds(), nil
				},
			},
			"rto": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Float),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(gqlFibNexthopRttRecord).RTO().Seconds(), nil
				},
			},
		},
	})
	fib.GqlEntryType.Object.AddFieldConfig("nexthopRtts", &graphql.Field{
		Description: "FIB nexthops and their RTT measurements in a forwarding thread.",
		Type:        gqlserver.NewNonNullList(GqlFibNexthopRttType),
		Args: graphql.FieldConfigArgument{
			"fwd": &graphql.ArgumentConfig{
				Description: "Forwarding thread ID. Default is the result of NDT lookup with FIB entry name.",
				Type:        graphql.ID,
			},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
			entry := p.Source.(fib.Entry)
			var fwd *Fwd
			if fwdID := p.Args["fwd"]; fwdID != nil {
				if fwd = GqlFwdType.Retrieve(fwdID.(string)); fwd == nil {
					return nil, errors.New("fwd not found")
				}
			} else if GqlDataPlane == nil {
				return nil, errNoGqlDataPlane
			} else if _, fwdIndex := GqlDataPlane.ndt.Lookup(entry.Name); int(fwdIndex) >= len(GqlDataPlane.fwds) {
				return nil, errors.New("cannot determine forwarding thread from NDT")
			} else {
				fwd = GqlDataPlane.fwds[fwdIndex]
			}

			rtts := entry.NexthopRtts(fwd)
			var list []gqlFibNexthopRttRecord
			for _, nh := range entry.Nexthops {
				list = append(list, gqlFibNexthopRttRecord{
					RttEstimator: rtts[nh],
					Face:         iface.Get(nh),
					Fwd:          fwd,
				})
			}
			return list, nil
		},
	})

	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "DiskStore counters in forwarder data plane.",
		Parent:       GqlDataPlaneType,
		Name:         "diskCounters",
		Subscription: "fwDiskCounters",
		Find: func(p graphql.ResolveParams) (source any, enders []any, e error) {
			return GqlDataPlane, nil, nil
		},
		Type: disk.GqlStoreCountersType,
		Read: func(p graphql.ResolveParams) (any, error) {
			dp := p.Source.(*DataPlane)
			if dp.fwdisk == nil {
				return nil, nil
			}
			return dp.fwdisk.store.Counters(), nil
		},
	})
}
