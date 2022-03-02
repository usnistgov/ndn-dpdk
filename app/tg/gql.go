package tg

import (
	"errors"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/app/fileserver"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go4.org/must"
)

var (
	// GqlCreateEnabled allows creating traffic generator instances via GraphQL.
	GqlCreateEnabled = false

	errGqlDisabled = errors.New("traffic generator not activated")
)

// GraphQL types.
var (
	GqlTrafficGenNodeType *gqlserver.NodeType
	GqlTrafficGenType     *graphql.Object
	GqlCountersType       *graphql.Object
)

func init() {
	retrieve := func(id iface.ID) interface{} { return Get(id) }
	GqlTrafficGenNodeType = tggql.NewNodeType("Tg", (*TrafficGen)(nil), &retrieve)
	GqlTrafficGenNodeType.Delete = func(source interface{}) error {
		return source.(*TrafficGen).Close()
	}
	GqlTrafficGenType = graphql.NewObject(GqlTrafficGenNodeType.Annotate(graphql.ObjectConfig{
		Name: "TrafficGen",
		Fields: tggql.CommonFields(graphql.Fields{
			"producer": &graphql.Field{
				Description: "Producer module.",
				Type:        tgproducer.GqlProducerType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					return gen.Producer(), nil
				},
			},
			"fileServer": &graphql.Field{
				Description: "File server module.",
				Type:        fileserver.GqlServerType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					return gen.FileServer(), nil
				},
			},
			"consumer": &graphql.Field{
				Description: "Consumer module.",
				Type:        tgconsumer.GqlConsumerType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					return gen.Consumer(), nil
				},
			},
			"fetcher": &graphql.Field{
				Description: "Fetcher module.",
				Type:        fetch.GqlFetcherType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					gen := p.Source.(*TrafficGen)
					return gen.Fetcher(), nil
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
				Type:        tgproducer.GqlConfigInput,
			},
			"fileServer": &graphql.ArgumentConfig{
				Description: "File server configuration.",
				Type:        fileserver.GqlConfigInput,
			},
			"consumer": &graphql.ArgumentConfig{
				Description: "Consumer configuration.",
				Type:        tgconsumer.GqlConfigInput,
			},
			"fetcher": &graphql.ArgumentConfig{
				Description: "Fetcher configuration.",
				Type:        fetch.GqlConfigInput,
			},
		},
		Type: graphql.NewNonNull(GqlTrafficGenType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if !GqlCreateEnabled {
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
					producer := p.Source.(*TrafficGen).Producer()
					if producer == nil {
						return nil, nil
					}
					return producer.Counters(), nil
				},
			},
			"consumer": &graphql.Field{
				Type: tgconsumer.GqlCountersType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					consumer := p.Source.(*TrafficGen).Consumer()
					if consumer == nil {
						return nil, nil
					}
					return consumer.Counters(), nil
				},
			},
		},
	})

	gqlserver.AddCounters(&gqlserver.Counters{
		Description:  "Obtain traffic generator counters.",
		Type:         GqlCountersType,
		Value:        (*TrafficGen)(nil),
		Subscription: "tgCounters",
		FindArgs: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "Traffic generator ID.",
				Type:        gqlserver.NonNullID,
			},
		},
		Find: func(p graphql.ResolveParams) (root interface{}, enders []interface{}, e error) {
			id := p.Args["id"].(string)
			var gen *TrafficGen
			if e := gqlserver.RetrieveNodeOfType(GqlTrafficGenNodeType, id, &gen); e != nil {
				return nil, nil, e
			}
			return gen, []interface{}{gen.exit}, nil
		},
		Read: func(p graphql.ResolveParams) (interface{}, error) {
			return p.Source.(*TrafficGen), nil
		},
	})
}
