package strategycode

import (
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraphQL types.
var (
	GqlStrategyNodeType *gqlserver.NodeType
	GqlStrategyType     *graphql.Object
)

func init() {
	GqlStrategyNodeType = gqlserver.NewNodeType((*Strategy)(nil))
	GqlStrategyNodeType.Retrieve = func(id string) (interface{}, error) {
		nid, e := strconv.Atoi(id)
		if e != nil {
			return nil, e
		}
		return Get(nid), nil
	}
	GqlStrategyNodeType.Delete = func(source interface{}) error {
		strategy := source.(*Strategy)
		return strategy.Close()
	}

	GqlStrategyType = graphql.NewObject(GqlStrategyNodeType.Annotate(graphql.ObjectConfig{
		Name: "Strategy",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Numeric strategy code identifier.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					strategy := p.Source.(*Strategy)
					return int(strategy.ID()), nil
				},
			},
			"name": &graphql.Field{
				Description: "Short name.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					strategy := p.Source.(*Strategy)
					return strategy.Name(), nil
				},
			},
		},
	}))
	GqlStrategyNodeType.Register(GqlStrategyType)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "strategies",
		Description: "List of strategies.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlStrategyType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return List(), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "loadStrategy",
		Description: "Upload a strategy ELF program.",
		Args: graphql.FieldConfigArgument{
			"name": &graphql.ArgumentConfig{
				Description: "Short name.",
				Type:        gqlserver.NonNullString,
			},
			"elf": &graphql.ArgumentConfig{
				Description: "ELF program in base64 format.",
				Type:        graphql.NewNonNull(gqlserver.Bytes),
			},
		},
		Type: graphql.NewNonNull(GqlStrategyType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			name := p.Args["name"].(string)
			elf := p.Args["elf"].([]byte)
			return Load(name, elf)
		},
	})
}
