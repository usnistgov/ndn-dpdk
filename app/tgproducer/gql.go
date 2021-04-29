package tgproducer

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
)

// GraphQL types.
var (
	GqlProducerNodeType *gqlserver.NodeType
	GqlProducerType     *graphql.Object
)

func init() {
	GqlProducerNodeType = tggql.NewNodeType((*Producer)(nil), "Producer")
	GqlProducerType = graphql.NewObject(GqlProducerNodeType.Annotate(graphql.ObjectConfig{
		Name: "TgProducer",
		Fields: tggql.CommonFields(graphql.Fields{
			"patterns": &graphql.Field{
				Description: "Traffic patterns.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					producer := p.Source.(*Producer)
					return producer.Patterns(), nil
				},
			},
			"counters": &graphql.Field{
				Description: "Counters.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					producer := p.Source.(*Producer)
					return producer.Counters(), nil
				},
			},
		}),
	}))
	GqlProducerNodeType.Register(GqlProducerType)
	tggql.AddFaceField("tgProducer", "Traffic generator producer on this face.", "Producer", GqlProducerType)
}
