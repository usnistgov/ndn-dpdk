package tgconsumer

import (
	"reflect"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// GqlRetrieveByFaceID returns *Consumer associated with a face.
// It is assigned during package tg initialization.
var GqlRetrieveByFaceID func(id iface.ID) *Consumer

// GraphQL types.
var (
	GqlPatternInput        *graphql.InputObject
	GqlConfigInput         *graphql.InputObject
	GqlPatternCountersType *graphql.Object
	GqlCountersType        *graphql.Object
	GqlConsumerType        *gqlserver.NodeType[*Consumer]
)

func init() {
	GqlPatternInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgcPatternInput",
		Description: "Traffic generator consumer pattern definition.",
		Fields: gqlserver.BindInputFields[Pattern](gqlserver.FieldTypes{
			reflect.TypeFor[ndn.Name]():                gqlserver.NonNullString,
			reflect.TypeFor[nnduration.Milliseconds](): nnduration.GqlMilliseconds,
			reflect.TypeFor[ndni.DataGenConfig]():      ndni.GqlDataGenInput,
		}),
	})
	GqlConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgcConfigInput",
		Description: "Traffic generator consumer config.",
		Fields: gqlserver.BindInputFields[Config](gqlserver.FieldTypes{
			reflect.TypeFor[iface.PktQueueConfig]():   iface.GqlPktQueueInput,
			reflect.TypeFor[nnduration.Nanoseconds](): nnduration.GqlNanoseconds,
			reflect.TypeFor[Pattern]():                GqlPatternInput,
		}),
	})

	GqlPatternCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "TgcPatternCounters",
		Fields: gqlserver.BindFields[PatternCounters](gqlserver.FieldTypes{
			reflect.TypeFor[runningstat.Snapshot](): runningstat.GqlSnapshotType,
		}),
	})
	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "TgcCounters",
		Fields: gqlserver.BindFields[Counters](gqlserver.FieldTypes{
			reflect.TypeFor[runningstat.Snapshot](): runningstat.GqlSnapshotType,
			reflect.TypeFor[PatternCounters]():      GqlPatternCountersType,
		}),
	})

	GqlConsumerType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "TgConsumer",
		Fields: tggql.CommonFields(graphql.Fields{
			"patterns": &graphql.Field{
				Description: "Traffic patterns.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(*Consumer).Patterns(), nil
				},
			},
			"counters": &graphql.Field{
				Description: "Counters.",
				Type:        graphql.NewNonNull(GqlCountersType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					return p.Source.(*Consumer).Counters(), nil
				},
			},
		}),
	}, tggql.NodeConfig(&GqlRetrieveByFaceID))
}
