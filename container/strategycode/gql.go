package strategycode

import (
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraghQL types.
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
}
