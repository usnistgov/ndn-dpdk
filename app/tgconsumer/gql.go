package tgconsumer

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// GqlRetrieveByFaceID returns *Consumer associated with a face.
// It is assigned during package tg initialization.
var GqlRetrieveByFaceID func(id iface.ID) interface{}

// GraphQL types.
var (
	GqlPatternInput     *graphql.InputObject
	GqlConsumerNodeType *gqlserver.NodeType
	GqlConsumerType     *graphql.Object
)

func init() {
	GqlPatternInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgcPatternInput",
		Description: "Traffic generator consumer pattern definition.",
		Fields: graphql.InputObjectConfigFieldMap{
			"weight": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
			"prefix": &graphql.InputObjectFieldConfig{
				Type: gqlserver.NonNullString,
			},
			"canBePrefix": &graphql.InputObjectFieldConfig{
				Type: graphql.Boolean,
			},
			"mustBeFresh": &graphql.InputObjectFieldConfig{
				Type: graphql.Boolean,
			},
			"interestLifetime": &graphql.InputObjectFieldConfig{
				Type: nnduration.GqlMilliseconds,
			},
			"hopLimit": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
			"seqNumOffset": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
		},
	})

	GqlConsumerNodeType = tggql.NewNodeType((*Consumer)(nil), &GqlRetrieveByFaceID)
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
}
