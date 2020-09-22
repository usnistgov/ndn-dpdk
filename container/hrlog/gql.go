package hrlog

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraphQL types.
var (
	GqlCollectJobNodeType *gqlserver.NodeType
	GqlCollectJobType     *graphql.Object
)

func init() {
	GqlCollectJobNodeType = gqlserver.NewNodeType((*Collector)(nil))
	GqlCollectJobNodeType.GetID = func(source interface{}) string {
		c := source.(*Collector)
		return c.cfg.Filename
	}
	GqlCollectJobNodeType.Retrieve = func(id string) (interface{}, error) {
		collectorLock.Lock()
		defer collectorLock.Unlock()
		return collectorMap[id], nil
	}
	GqlCollectJobNodeType.Delete = func(source interface{}) error {
		c := source.(*Collector)
		return c.Stop()
	}

	GqlCollectJobType = graphql.NewObject(GqlCollectJobNodeType.Annotate(graphql.ObjectConfig{
		Name: "HrlogCollectJob",
		Fields: graphql.Fields{
			"filename": &graphql.Field{
				Type:        gqlserver.NonNullString,
				Description: "Filename.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					c := p.Source.(*Collector)
					return c.cfg.Filename, nil
				},
			},
		},
	}))
	GqlCollectJobNodeType.Register(GqlCollectJobType)

	gqlserver.AddMutation(&graphql.Field{
		Name:        "collectHrlog",
		Description: "Start hrlog collection.",
		Args: graphql.FieldConfigArgument{
			"filename": &graphql.ArgumentConfig{
				Type: gqlserver.NonNullString,
			},
			"count": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
		},
		Type: GqlCollectJobType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			cfg := Config{
				Filename: p.Args["filename"].(string),
			}
			if count, ok := p.Args["count"]; ok {
				cfg.Count = count.(int)
			}
			return Start(cfg)
		},
	})
}
