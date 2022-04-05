package strategycode

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraphQL types.
var (
	GqlStrategyType *gqlserver.NodeType[*Strategy]
)

func init() {
	GqlStrategyType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "Strategy",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Numeric strategy code identifier.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					sc := p.Source.(*Strategy)
					return int(sc.ID()), nil
				},
			},
			"name": &graphql.Field{
				Description: "Short name.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					sc := p.Source.(*Strategy)
					return sc.Name(), nil
				},
			},
		},
	}, gqlserver.NodeConfig[*Strategy]{
		RetrieveInt: Get,
		Delete: func(sc *Strategy) error {
			sc.Unref()
			return nil
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "strategies",
		Description: "List of strategies.",
		Type:        gqlserver.NewNonNullList(GqlStrategyType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
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
		Type: graphql.NewNonNull(GqlStrategyType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			name := p.Args["name"].(string)
			elf := p.Args["elf"].([]byte)
			return Load(name, elf)
		},
	})
}
