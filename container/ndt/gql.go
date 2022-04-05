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
	//lint:ignore ST1005 'Index' is a field name
	errNoIndex = errors.New("Index is unspecified")
)

// GraphQL types.
var (
	GqlConfigType *graphql.Object
	GqlEntryType  *graphql.Object
)

func init() {
	GqlConfigType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "NdtConfig",
		Fields: gqlserver.BindFields[Config](nil),
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "ndtConfig",
		Description: "NDT configuration.",
		Type:        graphql.NewNonNull(GqlConfigType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if GqlNdt == nil {
				return nil, errNoGqlNdt
			}

			return GqlNdt.Config(), nil
		},
	})

	GqlEntryType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "NdtEntry",
		Fields: gqlserver.BindFields[Entry](nil),
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "ndt",
		Description: "List of NDT entries.",
		Type:        gqlserver.NewNonNullList(GqlEntryType),
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{
				Description: "Filter by name.",
				Type:        ndni.GqlNameType,
			},
		},
		Resolve: func(p graphql.ResolveParams) (any, error) {
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
		Resolve: func(p graphql.ResolveParams) (any, error) {
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
