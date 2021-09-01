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
var GqlRetrieveByFaceID func(id iface.ID) interface{}

// GraphQL types.
var (
	GqlPatternInput        *graphql.InputObject
	GqlConfigInput         *graphql.InputObject
	GqlPatternCountersType *graphql.Object
	GqlCountersType        *graphql.Object
	GqlConsumerNodeType    *gqlserver.NodeType
	GqlConsumerType        *graphql.Object
)

func init() {
	GqlPatternInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgcPatternInput",
		Description: "Traffic generator consumer pattern definition.",
		Fields: gqlserver.BindInputFields(Pattern{}, gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}):                 gqlserver.NonNullString,
			reflect.TypeOf(nnduration.Milliseconds(0)): nnduration.GqlMilliseconds,
			reflect.TypeOf(ndni.DataGenConfig{}):       ndni.GqlDataGenInput,
		}),
	})
	GqlConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgConsumerConfigInput",
		Description: "Traffic generator consumer config.",
		Fields: gqlserver.BindInputFields(Config{}, gqlserver.FieldTypes{
			reflect.TypeOf(iface.PktQueueConfig{}):    iface.GqlPktQueueInput,
			reflect.TypeOf(nnduration.Nanoseconds(0)): nnduration.GqlNanoseconds,
			reflect.TypeOf(Pattern{}):                 GqlPatternInput,
		}),
	})

	GqlPatternCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "TgcPatternCounters",
		Fields: gqlserver.BindFields(PatternCounters{}, gqlserver.FieldTypes{
			reflect.TypeOf(RttCounters{}): runningstat.GqlSnapshotType,
		}),
	})
	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "TgcCounters",
		Fields: gqlserver.BindFields(Counters{}, gqlserver.FieldTypes{
			reflect.TypeOf(RttCounters{}):     runningstat.GqlSnapshotType,
			reflect.TypeOf(PatternCounters{}): GqlPatternCountersType,
		}),
	})

	GqlConsumerNodeType = tggql.NewNodeType("Tgc", (*Consumer)(nil), &GqlRetrieveByFaceID)
	GqlConsumerType = graphql.NewObject(GqlConsumerNodeType.Annotate(graphql.ObjectConfig{
		Name: "TgConsumer",
		Fields: tggql.CommonFields(graphql.Fields{
			"patterns": &graphql.Field{
				Description: "Traffic patterns.",
				Type:        gqlserver.NonNullJSON,
				Resolve:     gqlserver.MethodResolver("Patterns"),
			},
			"counters": &graphql.Field{
				Description: "Counters.",
				Type:        graphql.NewNonNull(GqlCountersType),
				Resolve:     gqlserver.MethodResolver("Counters"),
			},
		}),
	}))
	GqlConsumerNodeType.Register(GqlConsumerType)
}
