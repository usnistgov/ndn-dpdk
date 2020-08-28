package ndt

import (
	"errors"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var (
	// GqlNdt is the NDT instance accessible via GraphQL.
	GqlNdt *Ndt

	errNoGqlNdt = errors.New("NDT unavailable")
	errNoIndex  = errors.New("Index is unspecified")
)

// GraghQL types.
var (
	GqlConfigType *graphql.Object
	GqlEntryType  *graphql.Object
)

func init() {
	GqlConfigType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "NdtConfig",
		Fields: graphql.BindFields(Config{}),
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "ndtConfig",
		Description: "NDT configuration.",
		Type:        graphql.NewNonNull(GqlConfigType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlNdt == nil {
				return nil, errNoGqlNdt
			}

			return GqlNdt.Config(), nil
		},
	})

	GqlEntryType = graphql.NewObject(graphql.ObjectConfig{
		Name: "NdtEntry",
		Fields: graphql.Fields{
			"index": &graphql.Field{
				Description: "Entry index.",
				Type:        gqlserver.NonNullInt,
			},
			"value": &graphql.Field{
				Description: "Entry value, aka forwarding thread index.",
				Type:        gqlserver.NonNullInt,
			},
			"hits": &graphql.Field{
				Description: "Hit counter value, wrapping at uint32 limit.",
				Type:        gqlserver.NonNullInt,
			},
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "ndt",
		Description: "List of NDT entries.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlEntryType)),
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{
				Type:        ndni.GqlNameType,
				Description: "Filter by name.",
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlNdt == nil {
				return nil, errNoGqlNdt
			}

			if name, ok := p.Args["name"].(ndn.Name); ok {
				entry := GqlNdt.Get(GqlNdt.IndexOfName(name))
				return []Entry{entry}, nil
			}

			return GqlNdt.List(), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "updateNdt",
		Description: "Update NDT entry.",
		Args: graphql.FieldConfigArgument{
			"index": &graphql.ArgumentConfig{
				Description: "Entry index. Either 'index' or 'name' is required; if both are specified, 'index' is preferred.",
				Type:        graphql.Int,
			},
			"name": &graphql.ArgumentConfig{
				Description: "Name to derive index. Either 'index' or 'name' is required; if both are specified, 'index' is preferred.",
				Type:        ndni.GqlNameType,
			},
			"value": &graphql.ArgumentConfig{
				Description: "Entry value.",
				Type:        gqlserver.NonNullInt,
			},
		},
		Type: graphql.NewNonNull(GqlEntryType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlNdt == nil {
				return nil, errNoGqlNdt
			}

			var index uint64
			if i, ok := p.Args["index"].(int); ok {
				index = GqlNdt.IndexOfHash(uint64(i))
			} else if name, ok := p.Args["name"].(ndn.Name); ok {
				index = GqlNdt.IndexOfName(name)
			} else {
				return nil, errNoIndex
			}

			GqlNdt.Update(index, uint8(p.Args["value"].(int)))
			return GqlNdt.Get(index), nil
		},
	})
}
