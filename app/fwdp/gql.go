package fwdp

import (
	"errors"
	"reflect"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/cs"
	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/rttest"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

var (
	// GqlDataPlane is the DataPlane instance accessible via GraphQL.
	GqlDataPlane *DataPlane

	errNoGqlDataPlane = errors.New("DataPlane unavailable")
)

func gqlRetrieveDispatch[T DispatchThread](id int) (th T) {
	if GqlDataPlane == nil {
		return
	}
	if id < 0 || id >= len(GqlDataPlane.dispatch) {
		return
	}
	th, _ = GqlDataPlane.dispatch[id].(T)
	return
}

type gqlFibNexthopRttRecord struct {
	*rttest.RttEstimator
	Face iface.Face `json:"face"`
	Fwd  *Fwd       `json:"fwd"`
}

// GraphQL types.
var (
	GqlDispatchThreadInterface *gqlserver.Interface
	GqlInputType               *gqlserver.NodeType[*Input]
	GqlCryptoType              *gqlserver.NodeType[*Crypto]
	GqlDiskType                *gqlserver.NodeType[*Disk]
	GqlFwdType                 *gqlserver.NodeType[*Fwd]
	GqlDataPlaneType           *graphql.Object
	GqlDispatchCountersType    *graphql.Object
	GqlFwdCountersType         *graphql.Object
	GqlFibNexthopRttType       *graphql.Object
)

func init() {
	GqlDispatchThreadInterface = gqlserver.NewInterface(graphql.InterfaceConfig{
		Name: "FwDispatchThread",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Dispatch thread index.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(DispatchThread).DispatchThreadID(), nil
				},
			},
			"worker": ealthread.GqlWithWorker(nil),
		},
	})

	GqlInputType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "FwInput",
		Fields: GqlDispatchThreadInterface.CopyFieldsTo(graphql.Fields{
			"worker": ealthread.GqlWithWorker(func(p graphql.ResolveParams) ealthread.Thread {
				input := p.Source.(*Input)
				return input.rxl
			}),
			"rxGroups": &graphql.Field{
				Description: "RX groups.",
				Type:        gqlserver.NewListNonNullBoth(iface.GqlRxGroupInterface.Interface),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					input := p.Source.(*Input)
					return input.rxl.List(), nil
				},
			},
		}),
	}, gqlserver.NodeConfig[*Input]{
		RetrieveInt: gqlRetrieveDispatch[*Input],
	})
	gqlserver.ImplementsInterface[*Input](GqlInputType.Object, GqlDispatchThreadInterface)

	GqlCryptoType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name:   "FwCrypto",
		Fields: GqlDispatchThreadInterface.CopyFieldsTo(nil),
	}, gqlserver.NodeConfig[*Crypto]{
		RetrieveInt: gqlRetrieveDispatch[*Crypto],
	})
	gqlserver.ImplementsInterface[*Input](GqlCryptoType.Object, GqlDispatchThreadInterface)

	GqlDiskType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name:   "FwDisk",
		Fields: GqlDispatchThreadInterface.CopyFieldsTo(nil),
	}, gqlserver.NodeConfig[*Disk]{
		RetrieveInt: gqlRetrieveDispatch[*Disk],
	})
	gqlserver.ImplementsInterface[*Disk](GqlDiskType.Object, GqlDispatchThreadInterface)

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
				Type:        gqlserver.NewListNonNullBoth(GqlInputType.Object),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					dp := p.Source.(*DataPlane)
					return dp.fwis, nil
				},
			},
			"cryptos": &graphql.Field{
				Description: "Crypto helper threads.",
				Type:        gqlserver.NewListNonNullBoth(GqlCryptoType.Object),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					dp := p.Source.(*DataPlane)
					return dp.fwcs, nil
				},
			},
			"disks": &graphql.Field{
				Description: "Disk service threads.",
				Type:        gqlserver.NewListNonNullBoth(GqlDiskType.Object),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					dp := p.Source.(*DataPlane)
					list := []*Disk{}
					if dp.fwdisk != nil {
						list = append(list, dp.fwdisk)
					}
					return list, nil
				},
			},
			"fwds": &graphql.Field{
				Description: "Forwarding threads.",
				Type:        gqlserver.NewListNonNullBoth(GqlFwdType.Object),
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

	GqlDispatchCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FwDispatchCounters",
		Fields: gqlserver.BindFields[DispatchCounters](nil),
	})
	const dispatchCntField = "dispatchCounters"
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "Packets dispatched from input/crypto/disk thread to each forwarding thread in forwarder data plane.",
		Parent:       GqlInputType.Object,
		Name:         dispatchCntField,
		Subscription: "fwDispatchCounters",
		FindArgs: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "Input/crypto/disk thread ID.",
				Type:        gqlserver.NonNullID,
			},
		},
		Find: func(p graphql.ResolveParams) (source any, enders []any, e error) {
			obj, _ := gqlserver.RetrieveNode(p.Args["id"].(string))
			source, _ = obj.(DispatchThread)
			return
		},
		Type: GqlDispatchCountersType,
		Read: func(p graphql.ResolveParams) (any, error) {
			th := p.Source.(DispatchThread)
			return ReadDispatchCounters(th, len(GqlDataPlane.fwds)), nil
		},
	})
	for _, object := range []*graphql.Object{GqlCryptoType.Object, GqlDiskType.Object} {
		object.AddFieldConfig(dispatchCntField, gqlserver.FieldDefToField(GqlInputType.Object.Fields()[dispatchCntField]))
	}

	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "Forwarder DiskStore counters.",
		Parent:       GqlDiskType.Object,
		Name:         "storeCounters",
		Subscription: "fwDiskCounters",
		Find: func(graphql.ResolveParams) (source any, enders []any, e error) {
			if GqlDataPlane == nil {
				return nil, nil, nil
			}
			return GqlDataPlane.fwdisk, nil, nil
		},
		Type: disk.GqlStoreCountersType,
		Read: func(p graphql.ResolveParams) (any, error) {
			fwdisk := p.Source.(*Disk)
			return fwdisk.store.Counters(), nil
		},
	})

	GqlFwdCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "FwFwdCounters",
		Fields: gqlserver.BindFields[FwdCounters](gqlserver.FieldTypes{
			reflect.TypeOf(runningstat.Snapshot{}): runningstat.GqlSnapshotType,
		}),
	})
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
		Type:        gqlserver.NewListNonNullBoth(GqlFibNexthopRttType),
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
}
