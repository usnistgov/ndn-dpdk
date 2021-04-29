package tgconsumer

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraphQL types.
var (
	GqlConsumerNodeType *gqlserver.NodeType
	GqlConsumerType     *graphql.Object
)

func init() {
	GqlConsumerNodeType = tggql.NewNodeType((*Consumer)(nil), "Consumer")
	GqlConsumerType = graphql.NewObject(GqlConsumerNodeType.Annotate(graphql.ObjectConfig{
		Name: "TgConsumer",
		Fields: tggql.CommonFields(graphql.Fields{
			"patterns": &graphql.Field{
				Description: "Traffic patterns.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					consumer := p.Source.(*Consumer)
					return consumer.Patterns(), nil
				},
			},
			"counters": &graphql.Field{
				Description: "Counters.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					consumer := p.Source.(*Consumer)
					return consumer.Counters(), nil
				},
			},
		}),
	}))
	GqlConsumerNodeType.Register(GqlConsumerType)
	tggql.AddFaceField("tgConsumer", "Traffic generator consumer on this face.", "Consumer", GqlConsumerType)
}
