package fib

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/strategycode"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var (
	// GqlFib is the FIB instance accessible via GraphQL.
	GqlFib *Fib

	errNoGqlFib = errors.New("FIB unavailable")

	// GqlDefaultStrategy is the default strategy when inserted a FIB entry via GraphQL.
	GqlDefaultStrategy *strategycode.Strategy
)

// GraphQL types.
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
		if GqlFib == nil {
			return errNoGqlFib
		}
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
				Type:        gqlserver.NewNonNullList(iface.GqlFaceType, true),
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
		Type:        gqlserver.NewNonNullList(GqlEntryType),
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

	gqlserver.AddMutation(&graphql.Field{
		Name:        "insertFibEntry",
		Description: "Insert or replace a FIB entry.",
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{
				Description: "Entry name.",
				Type:        graphql.NewNonNull(ndni.GqlNameType),
			},
			"nexthops": &graphql.ArgumentConfig{
				Description: "FIB nexthops.",
				Type:        gqlserver.NewNonNullList(gqlserver.NonNullID),
			},
			"strategy": &graphql.ArgumentConfig{
				Description: "Forwarding strategy.",
				Type:        graphql.ID,
			},
		},
		Type: graphql.NewNonNull(GqlEntryType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlFib == nil {
				return nil, errNoGqlFib
			}

			var entry fibdef.Entry
			entry.Name = p.Args["name"].(ndn.Name)
			for i, nh := range p.Args["nexthops"].([]interface{}) {
				face, e := gqlserver.RetrieveNodeOfType(iface.GqlFaceNodeType, nh)
				if face == nil || e != nil {
					return nil, fmt.Errorf("nexthops[%d] not found: %w", i, e)
				}
				entry.Nexthops = append(entry.Nexthops, face.(iface.Face).ID())
			}

			if strategy, ok := p.Args["strategy"].(string); ok {
				strategyCode, e := gqlserver.RetrieveNodeOfType(strategycode.GqlStrategyNodeType, strategy)
				if strategyCode == nil || e != nil {
					return nil, fmt.Errorf("strategy not found: %w", e)
				}
				entry.Strategy = strategyCode.(*strategycode.Strategy).ID()
			} else if GqlDefaultStrategy != nil {
				entry.Strategy = GqlDefaultStrategy.ID()
			}

			if e := GqlFib.Insert(entry); e != nil {
				return nil, e
			}
			return *GqlFib.Find(entry.Name), nil
		},
	})
}
