package tg

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

var (
	// GqlTrafficGen is the TrafficGen instance accessible via GraphQL.
	GqlTrafficGen *TrafficGen

	errNoGqlTrafficGen = errors.New("TrafficGen unavailable")
)

// GraphQL types.
var (
	GqlProducerNodeType *gqlserver.NodeType
	GqlProducerType     *graphql.Object
	GqlConsumerNodeType *gqlserver.NodeType
	GqlConsumerType     *graphql.Object
	GqlTrafficGenType   *graphql.Object
)

func init() {
	GqlProducerNodeType = gqlserver.NewNodeType((*tgproducer.Producer)(nil))
	GqlProducerNodeType.GetID = func(source interface{}) string {
		producer := source.(*tgproducer.Producer)
		return fmt.Sprintf("%d:%d", producer.Face().ID(), producer.Index)
	}
	GqlProducerNodeType.Retrieve = func(id string) (interface{}, error) {
		if GqlTrafficGen == nil {
			return nil, errNoGqlTrafficGen
		}
		var faceID, index int
		if _, e := fmt.Scanf("%d:%d", &faceID, &index); e != nil {
			return nil, nil
		}
		task := GqlTrafficGen.Task(iface.ID(faceID))
		if task == nil || index < 0 || index >= len(task.Producers) {
			return nil, nil
		}
		return task.Producers[index], nil
	}

	GqlProducerType = graphql.NewObject(GqlProducerNodeType.Annotate(graphql.ObjectConfig{
		Name: "TgProducer",
		Fields: graphql.Fields{
			"worker": ealthread.GqlWithWorker(nil),
			"face": &graphql.Field{
				Description: "Face.",
				Type:        graphql.NewNonNull(iface.GqlFaceType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					producer := p.Source.(*tgproducer.Producer)
					return producer.Face(), nil
				},
			},
			"patterns": &graphql.Field{
				Description: "Traffic patterns.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					producer := p.Source.(*tgproducer.Producer)
					return producer.Patterns(), nil
				},
			},
			"counters": &graphql.Field{
				Description: "Counters.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					producer := p.Source.(*tgproducer.Producer)
					return producer.ReadCounters(), nil
				},
			},
		},
	}))
	GqlProducerNodeType.Register(GqlProducerType)

	iface.GqlFaceType.AddFieldConfig("tgProducers", &graphql.Field{
		Description: "Traffic generator producers on this face.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlProducerType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlTrafficGen == nil {
				return nil, nil
			}
			face := p.Source.(iface.Face)
			task := GqlTrafficGen.Task(face.ID())
			if task == nil {
				return nil, nil
			}
			return task.Producers, nil
		},
	})

	consumerFromID := func(id iface.ID, errNoTrafficGen error) (interface{}, error) {
		if GqlTrafficGen == nil {
			return nil, errNoTrafficGen
		}
		task := GqlTrafficGen.Task(id)
		if task == nil {
			return nil, nil
		}
		return task.Consumer, nil
	}

	GqlConsumerNodeType = gqlserver.NewNodeType((*tgconsumer.Consumer)(nil))
	GqlConsumerNodeType.GetID = func(source interface{}) string {
		consumer := source.(*tgconsumer.Consumer)
		return strconv.Itoa(int(consumer.Face().ID()))
	}
	GqlConsumerNodeType.Retrieve = func(id string) (interface{}, error) {
		i, e := strconv.Atoi(id)
		if e != nil {
			return nil, nil
		}
		return consumerFromID(iface.ID(i), errNoGqlTrafficGen)
	}

	GqlConsumerType = graphql.NewObject(GqlConsumerNodeType.Annotate(graphql.ObjectConfig{
		Name: "TgConsumer",
		Fields: graphql.Fields{
			"workerRx": ealthread.GqlWithWorker(func(p graphql.ResolveParams) ealthread.Thread {
				consumer := p.Source.(*tgconsumer.Consumer)
				return consumer.Rx
			}),
			"workerTx": ealthread.GqlWithWorker(func(p graphql.ResolveParams) ealthread.Thread {
				consumer := p.Source.(*tgconsumer.Consumer)
				return consumer.Tx
			}),
			"face": &graphql.Field{
				Description: "Face.",
				Type:        graphql.NewNonNull(iface.GqlFaceType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					consumer := p.Source.(*tgconsumer.Consumer)
					return consumer.Face(), nil
				},
			},
			"patterns": &graphql.Field{
				Description: "Traffic patterns.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					consumer := p.Source.(*tgconsumer.Consumer)
					return consumer.Patterns(), nil
				},
			},
			"counters": &graphql.Field{
				Description: "Counters.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					consumer := p.Source.(*tgconsumer.Consumer)
					return consumer.ReadCounters(), nil
				},
			},
		},
	}))
	GqlConsumerNodeType.Register(GqlConsumerType)

	iface.GqlFaceType.AddFieldConfig("tgConsumer", &graphql.Field{
		Description: "Traffic generator consumer on this face.",
		Type:        GqlConsumerType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if GqlTrafficGen == nil {
				return nil, nil
			}
			face := p.Source.(iface.Face)
			return consumerFromID(face.ID(), nil)
		},
	})

	GqlTrafficGenType = graphql.NewObject(graphql.ObjectConfig{
		Name: "TrafficGen",
		Fields: graphql.Fields{
			"producers": &graphql.Field{
				Description: "Producers.",
				Type:        gqlserver.NewNonNullList(GqlProducerType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					producers := []*tgproducer.Producer{}
					for _, task := range gen.Tasks {
						producers = append(producers, task.Producers...)
					}
					return producers, nil
				},
			},
			"consumers": &graphql.Field{
				Description: "Consumers.",
				Type:        gqlserver.NewNonNullList(GqlConsumerType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					consumers := []*tgconsumer.Consumer{}
					for _, task := range gen.Tasks {
						if task.Consumer != nil {
							consumers = append(consumers, task.Consumer)
						}
					}
					return consumers, nil
				},
			},
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "trafficgen",
		Description: "Traffic generator.",
		Type:        GqlTrafficGenType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return GqlTrafficGen, nil
		},
	})
}
