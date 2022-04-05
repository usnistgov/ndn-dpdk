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

	// GqlDefaultStrategy is the default strategy when inserting a FIB entry via GraphQL.
	GqlDefaultStrategy *strategycode.Strategy
)

// GraphQL types.
var (
	GqlEntryCountersType graphql.Type
	GqlEntryType         *gqlserver.NodeType[Entry]
)

func init() {
	GqlEntryCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FibEntryCounters",
		Fields: gqlserver.BindFields[fibdef.EntryCounters](nil),
	})

	GqlEntryType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "FibEntry",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Description: "Entry name.",
				Type:        graphql.NewNonNull(ndni.GqlNameType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					entry := p.Source.(Entry)
					return entry.Name, nil
				},
			},
			"nexthops": &graphql.Field{
				Description: "FIB nexthops. null indicates a deleted face.",
				Type:        gqlserver.NewNonNullList(iface.GqlFaceType.Object, true),
				Resolve: func(p graphql.ResolveParams) (any, error) {
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
				Type:        strategycode.GqlStrategyType.Object,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					entry := p.Source.(Entry)
					return strategycode.Get(entry.Strategy), nil
				},
			},
			"counters": &graphql.Field{
				Description: "Entry counters.",
				Type:        graphql.NewNonNull(GqlEntryCountersType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					entry := p.Source.(Entry)
					return entry.Counters(), nil
				},
			},
		},
	}, gqlserver.NodeConfig[Entry]{
		GetID: func(entry Entry) string {
			nameV, _ := entry.Name.MarshalBinary()
			return string(nameV)
		},
		Retrieve: func(id string) (entry Entry) {
			if GqlFib == nil {
				return
			}

			var name ndn.Name
			if e := name.UnmarshalBinary([]byte(id)); e != nil {
				return
			}

			if entryPtr := GqlFib.Find(name); entryPtr != nil {
				return *entryPtr
			}
			return
		},
		Delete: func(entry Entry) error {
			if GqlFib == nil {
				return errNoGqlFib
			}
			return GqlFib.Erase(entry.Name)
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "fib",
		Description: "List of FIB entries.",
		Type:        gqlserver.NewNonNullList(GqlEntryType.Object),
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{
				Description: "Filter by exact name.",
				Type:        ndni.GqlNameType,
			},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
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

	iface.GqlFaceType.Object.AddFieldConfig("fibEntries", &graphql.Field{
		Description: "FIB entries having this face as nexthop.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlEntryType.Object)),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if GqlFib == nil {
				return nil, nil
			}
			face := p.Source.(iface.Face)
			faceID := face.ID()

			var list []Entry
			for _, entry := range GqlFib.List() {
				if entry.HasNextHop(faceID) {
					list = append(list, entry)
				}
			}
			return list, nil
		},
	})

	strategycode.GqlStrategyType.Object.AddFieldConfig("fibEntries", &graphql.Field{
		Description: "FIB entries using this strategy.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlEntryType.Object)),
		Resolve: func(p graphql.ResolveParams) (any, error) {
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
			"params": &graphql.ArgumentConfig{
				Description: "Forwarding strategy parameters.",
				Type:        gqlserver.JSON,
			},
		},
		Type: graphql.NewNonNull(GqlEntryType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if GqlFib == nil {
				return nil, errNoGqlFib
			}

			var entry fibdef.Entry
			entry.Name = p.Args["name"].(ndn.Name)
			for i, nh := range p.Args["nexthops"].([]any) {
				face := iface.GqlFaceType.Retrieve(nh.(string))
				if face == nil {
					return nil, fmt.Errorf("nexthops[%d] not found", i)
				}
				entry.Nexthops = append(entry.Nexthops, face.ID())
			}

			sc := GqlDefaultStrategy
			if strategy, ok := p.Args["strategy"].(string); ok {
				sc = strategycode.GqlStrategyType.Retrieve(strategy)
			}
			if sc == nil {
				return nil, fmt.Errorf("strategy not found")
			}
			entry.Strategy = sc.ID()

			if params, ok := p.Args["params"].(map[string]any); ok {
				entry.Params = params
			}

			if e := GqlFib.Insert(entry); e != nil {
				return nil, e
			}
			return *GqlFib.Find(entry.Name), nil
		},
	})
}
