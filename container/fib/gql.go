package fib

import (
	"errors"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// GqlFib is the FIB instance accessible via GraphQL.
var GqlFib *Fib

var errNoGqlFib = errors.New("FIB unavailable")

// GraghQL types.
var (
	GqlEntryCountersType graphql.Type
	GqlEntryNodeType     *gqlserver.NodeType
	GqlEntryType         *graphql.Object
)

func init() {
	GqlEntryCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FibEntryCounters",
		Fields: graphql.BindFields(fibdef.EntryCounters{}),
	})

	GqlEntryNodeType = gqlserver.NewNodeType(Entry{})
	GqlEntryNodeType.GetID = func(source interface{}) string {
		entry := source.(Entry)
		return entry.Name.String()
	}
	GqlEntryNodeType.Retrieve = func(id string) (interface{}, error) {
		if GqlFib == nil {
			return nil, errNoGqlFib
		}
		name := ndn.ParseName(id)
		entry := GqlFib.Find(name)
		if entry == nil {
			return nil, nil
		}
		return *entry, nil
	}
	GqlEntryNodeType.Delete = func(source interface{}) error {
		entry := source.(Entry)
		return GqlFib.Erase(entry.Name)
	}

	GqlEntryType = graphql.NewObject(GqlEntryNodeType.Annotate(graphql.ObjectConfig{
		Name: "FibEntry",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Description: "Entry name.",
				Type:        graphql.NewNonNull(ndni.GqlNameType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					entry := p.Source.(Entry)
					return entry.Name, nil
				},
			},
			"nexthops": &graphql.Field{
				Description: "FIB nexthops. null indicates a deleted face.",
				Type:        graphql.NewList(iface.GqlFaceType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					entry := p.Source.(Entry)
					var list []iface.Face
					for _, nh := range entry.Nexthops {
						list = append(list, iface.Get(nh))
					}
					return list, nil
				},
			},
			"strategy": &graphql.Field{
				Description: "Forwarding strategy. null indicates a deleted strategy.",
				Type:        strategycode.GqlStrategyType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					entry := p.Source.(Entry)
					return strategycode.Get(entry.Strategy), nil
				},
			},
			"counters": &graphql.Field{
				Description: "Entry counters.",
				Type:        graphql.NewNonNull(GqlEntryCountersType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					entry := p.Source.(Entry)
					return entry.Counters(), nil
				},
			},
		},
	}))
	GqlEntryNodeType.Register(GqlEntryType)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "fib",
		Description: "List of FIB entries.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlEntryType)),
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{
				Type:        ndni.GqlNameType,
				Description: "Filter by exact name.",
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlFib == nil {
				return nil, errNoGqlFib
			}

			if name, ok := p.Args["name"].(ndn.Name); ok {
				var list []Entry
				if entry := GqlFib.Find(name); entry != nil {
					list = append(list, *entry)
				}
				return list, nil
			}

			return GqlFib.List(), nil
		},
	})

	iface.GqlFaceType.AddFieldConfig("fibEntries", &graphql.Field{
		Description: "FIB entries having this face as nexthop.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlEntryType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlFib == nil {
				return nil, nil
			}
			face := p.Source.(iface.Face)
			faceID := face.ID()

			var list []Entry
			for _, entry := range GqlFib.List() {
				hasNh := false
				for _, nh := range entry.Nexthops {
					if nh == faceID {
						hasNh = true
						break
					}
				}
				if hasNh {
					list = append(list, entry)
				}
			}
			return list, nil
		},
	})

	strategycode.GqlStrategyType.AddFieldConfig("fibEntries", &graphql.Field{
		Description: "FIB entries using this strategy.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlEntryType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlFib == nil {
				return nil, nil
			}
			strategy := p.Source.(*strategycode.Strategy)
			strategyID := strategy.ID()

			var list []Entry
			for _, entry := range GqlFib.List() {
				if entry.Strategy == strategyID {
					list = append(list, entry)
				}
			}
			return list, nil
		},
	})
}
