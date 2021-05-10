package runningstat

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraphQL types.
var (
	GqlSnapshotType *graphql.Object
)

func init() {
	GqlSnapshotType = graphql.NewObject(graphql.ObjectConfig{
		Name: "RunningStatSnapshot",
		Fields: graphql.Fields{
			"count": &graphql.Field{
				Type:    gqlserver.NonNullString,
				Resolve: gqlserver.MethodResolver("Count"),
			},
			"len": &graphql.Field{
				Type:    gqlserver.NonNullString,
				Resolve: gqlserver.MethodResolver("Len"),
			},
			"min": &graphql.Field{
				Type:    graphql.Float,
				Resolve: gqlserver.MethodResolver("Min"),
			},
			"max": &graphql.Field{
				Type:    graphql.Float,
				Resolve: gqlserver.MethodResolver("Max"),
			},
			"mean": &graphql.Field{
				Type:    graphql.Float,
				Resolve: gqlserver.MethodResolver("Mean"),
			},
			"variance": &graphql.Field{
				Type:    graphql.Float,
				Resolve: gqlserver.MethodResolver("Variance"),
			},
			"stdev": &graphql.Field{
				Type:    graphql.Float,
				Resolve: gqlserver.MethodResolver("Stdev"),
			},
			"m1": &graphql.Field{
				Type:    graphql.Float,
				Resolve: gqlserver.MethodResolver("M1"),
			},
			"m2": &graphql.Field{
				Type:    graphql.Float,
				Resolve: gqlserver.MethodResolver("M2"),
			},
		},
	})
}
