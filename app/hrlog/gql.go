package hrlog

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraphQL types.
var (
	GqlTaskNodeType *gqlserver.NodeType
	GqlTaskType     *graphql.Object
)

func init() {
	GqlTaskNodeType = gqlserver.NewNodeType((*Task)(nil))
	GqlTaskNodeType.GetID = func(source interface{}) string {
		c := source.(*Task)
		return c.cfg.Filename
	}
	GqlTaskNodeType.Retrieve = func(id string) (interface{}, error) {
		if TheWriter == nil {
			return nil, nil
		}
		task, _ := TheWriter.tasks.Load(id)
		return task, nil
	}
	GqlTaskNodeType.Delete = func(source interface{}) error {
		task := source.(*Task)
		return task.Stop()
	}

	GqlTaskType = graphql.NewObject(GqlTaskNodeType.Annotate(graphql.ObjectConfig{
		Name: "HrlogTask",
		Fields: graphql.Fields{
			"filename": &graphql.Field{
				Type:        gqlserver.NonNullString,
				Description: "Filename.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					task := p.Source.(*Task)
					return task.cfg.Filename, nil
				},
			},
		},
	}))
	GqlTaskNodeType.Register(GqlTaskType)

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
		Type: GqlTaskType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if TheWriter == nil {
				return nil, ErrDisabled
			}

			cfg := TaskConfig{
				Filename: p.Args["filename"].(string),
			}
			if count, ok := p.Args["count"]; ok {
				cfg.Count = count.(int)
			}
			return TheWriter.Submit(cfg)
		},
	})
}
