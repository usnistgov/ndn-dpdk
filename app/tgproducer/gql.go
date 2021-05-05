package tgproducer

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// GqlRetrieveByFaceID returns *Producer associated with a face.
// It is assigned during package tg initialization.
var GqlRetrieveByFaceID func(id iface.ID) interface{}

// GraphQL types.
var (
	GqlReplyInput       *graphql.InputObject
	GqlPatternInput     *graphql.InputObject
	GqlProducerNodeType *gqlserver.NodeType
	GqlProducerType     *graphql.Object
)

func init() {
	GqlReplyInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgpReplyInput",
		Description: "Traffic generator producer reply definition.",
		Fields: graphql.InputObjectConfigFieldMap{
			"weight": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
			"suffix": &graphql.InputObjectFieldConfig{
				Type: graphql.String,
			},
			"freshnessPeriod": &graphql.InputObjectFieldConfig{
				Type: nnduration.GqlMilliseconds,
			},
			"payloadLen": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
			"nack": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
			"timeout": &graphql.InputObjectFieldConfig{
				Type: graphql.Boolean,
			},
		},
	})
	GqlPatternInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgpPatternInput",
		Description: "Traffic generator producer pattern definition.",
		Fields: graphql.InputObjectConfigFieldMap{
			"prefix": &graphql.InputObjectFieldConfig{
				Type: gqlserver.NonNullString,
			},
			"replies": &graphql.InputObjectFieldConfig{
				Type: gqlserver.NewNonNullList(GqlReplyInput),
			},
		},
	})

	GqlProducerNodeType = tggql.NewNodeType((*Producer)(nil), &GqlRetrieveByFaceID)
	GqlProducerType = graphql.NewObject(GqlProducerNodeType.Annotate(graphql.ObjectConfig{
		Name:        "TgProducer",
		Description: "Traffic generator producer.",
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
}
