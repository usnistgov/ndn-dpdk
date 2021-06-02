package tg

import (
	"context"
	"errors"
	"time"

	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver/gqlsub"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
)

var (
	// GqlEnabled allows creating traffic generator instances via GraphQL.
	GqlEnabled = false

	errGqlDisabled = errors.New("traffic generator not activated")
)

// GraphQL types.
var (
	GqlProducerConfigInput *graphql.InputObject
	GqlConsumerConfigInput *graphql.InputObject
	GqlTrafficGenNodeType  *gqlserver.NodeType
	GqlTrafficGenType      *graphql.Object
	GqlCountersType        *graphql.Object
)

func init() {
	GqlProducerConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgProducerConfigInput",
		Description: "Traffic generator producer config.",
		Fields: graphql.InputObjectConfigFieldMap{
			"rxQueue": &graphql.InputObjectFieldConfig{
				Type: iface.GqlPktQueueInput,
			},
			"patterns": &graphql.InputObjectFieldConfig{
				Type: gqlserver.NewNonNullList(tgproducer.GqlPatternInput),
			},
			"nThreads": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
		},
	})
	GqlConsumerConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "TgConsumerConfigInput",
		Description: "Traffic generator consumer config.",
		Fields: graphql.InputObjectConfigFieldMap{
			"rxQueue": &graphql.InputObjectFieldConfig{
				Type: iface.GqlPktQueueInput,
			},
			"patterns": &graphql.InputObjectFieldConfig{
				Type: gqlserver.NewNonNullList(tgconsumer.GqlPatternInput),
			},
			"interval": &graphql.InputObjectFieldConfig{
				Type: graphql.NewNonNull(nnduration.GqlNanoseconds),
			},
		},
	})

	retrieve := func(id iface.ID) interface{} { return Get(id) }
	GqlTrafficGenNodeType = tggql.NewNodeType((*TrafficGen)(nil), &retrieve)
	GqlTrafficGenNodeType.Delete = func(source interface{}) error {
		return source.(*TrafficGen).Close()
	}
	GqlTrafficGenType = graphql.NewObject(GqlTrafficGenNodeType.Annotate(graphql.ObjectConfig{
		Name: "TrafficGen",
		Fields: tggql.CommonFields(graphql.Fields{
			"producer": &graphql.Field{
				Description: "Producer element.",
				Type:        tgproducer.GqlProducerType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					return gen.producer, nil
				},
			},
			"consumer": &graphql.Field{
				Description: "Consumer element.",
				Type:        tgconsumer.GqlConsumerType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					return gen.consumer, nil
				},
			},
			"fetcher": &graphql.Field{
				Description: "Fetcher element.",
				Type:        fetch.GqlFetcherType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					return gen.fetcher, nil
				},
			},
		}),
	}))
	GqlTrafficGenNodeType.Register(GqlTrafficGenType)
	iface.GqlFaceType.AddFieldConfig("trafficgen", &graphql.Field{
		Description: "Traffic generator operating on this face.",
		Type:        GqlTrafficGenType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			face := p.Source.(iface.Face)
			return Get(face.ID()), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "startTrafficGen",
		Description: "Create and start a traffic generator.",
		Args: graphql.FieldConfigArgument{
			"face": &graphql.ArgumentConfig{
				Description: "JSON object that satisfies the schema given in 'locator.schema.json'.",
				Type:        gqlserver.NonNullJSON,
			},
			"producer": &graphql.ArgumentConfig{
				Description: "Producer configuration.",
				Type:        GqlProducerConfigInput,
			},
			"consumer": &graphql.ArgumentConfig{
				Description: "Consumer configuration.",
				Type:        GqlConsumerConfigInput,
			},
			"fetcher": &graphql.ArgumentConfig{
				Description: "Fetcher configuration.",
				Type:        fetch.GqlConfigInput,
			},
		},
		Type: graphql.NewNonNull(GqlTrafficGenType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if !GqlEnabled {
				return nil, errGqlDisabled
			}

			var cfg Config
			if e := jsonhelper.Roundtrip(p.Args, &cfg, jsonhelper.DisallowUnknownFields); e != nil {
				return nil, e
			}

			gen, e := New(cfg)
			if e != nil {
				return nil, e
			}
			if e := gen.Launch(); e != nil {
				must.Close(gen)
				return nil, e
			}
			return gen, nil
		},
	})

	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "TgCounters",
		Fields: graphql.Fields{
			"producer": &graphql.Field{
				Type: tgproducer.GqlCountersType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					producer := p.Source.(*TrafficGen).producer
					if producer == nil {
						return nil, nil
					}
					return producer.Counters(), nil
				},
			},
			"consumer": &graphql.Field{
				Type: tgconsumer.GqlCountersType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					consumer := p.Source.(*TrafficGen).consumer
					if consumer == nil {
						return nil, nil
					}
					return consumer.Counters(), nil
				},
			},
		},
	})

	gqlserver.AddSubscription(&graphql.Field{
		Name:        "tgCounters",
		Description: "Obtain traffic generator counters.",
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "Traffic generator ID.",
				Type:        gqlserver.NonNullID,
			},
			"interval": &graphql.ArgumentConfig{
				Description: "Interval between updates.",
				Type:        nnduration.GqlNanoseconds,
			},
		},
		Type: GqlCountersType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return p.Info.RootValue.(*TrafficGen), nil
		},
	}, func(ctx context.Context, sub *graphqlws.Subscription, updates chan<- interface{}) {
		defer close(updates)

		id, ok := gqlsub.GetArg(sub, "id", graphql.ID).(string)
		if !ok {
			return
		}

		var gen *TrafficGen
		if e := gqlserver.RetrieveNodeOfType(GqlTrafficGenNodeType, id, &gen); e != nil || gen == nil {
			return
		}

		interval, ok := gqlsub.GetArg(sub, "interval", nnduration.GqlNanoseconds).(nnduration.Nanoseconds)
		if !ok {
			return
		}

		ticker := time.NewTicker(interval.Duration())
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-gen.exit:
				return
			case <-ticker.C:
				updates <- gen
			}
		}
	})
}
