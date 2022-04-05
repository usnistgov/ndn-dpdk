package tg

import (
	"errors"
	"reflect"

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
	GqlTrafficGenType *gqlserver.NodeType[*TrafficGen]
	GqlCountersType   *graphql.Object
)

func init() {
	retrieve := Get
	nc := tggql.NodeConfig(&retrieve)
	nc.Delete = func(gen *TrafficGen) error {
		return gen.Close()

	}
	GqlTrafficGenType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "TrafficGen",
		Fields: tggql.CommonFields(graphql.Fields{
			"producer": &graphql.Field{
				Description: "Producer module.",
				Type:        tgproducer.GqlProducerType.Object,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					gen := p.Source.(*TrafficGen)
					return gen.Producer(), nil
				},
			},
			"fileServer": &graphql.Field{
				Description: "File server module.",
				Type:        fileserver.GqlServerType.Object,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					gen := p.Source.(*TrafficGen)
					return gen.FileServer(), nil
				},
			},
			"consumer": &graphql.Field{
				Description: "Consumer module.",
				Type:        tgconsumer.GqlConsumerType.Object,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					gen := p.Source.(*TrafficGen)
					return gen.Consumer(), nil
				},
			},
			"fetcher": &graphql.Field{
				Description: "Fetcher module.",
				Type:        fetch.GqlFetcherType.Object,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					gen := p.Source.(*TrafficGen)
					return gen.Fetcher(), nil
				},
			},
		}),
	}, nc)

	iface.GqlFaceType.Object.AddFieldConfig("trafficgen", &graphql.Field{
		Description: "Traffic generator operating on this face.",
		Type:        GqlTrafficGenType.Object,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			face := p.Source.(iface.Face)
			return Get(face.ID()), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "startTrafficGen",
		Description: "Create and start a traffic generator.",
		Args: gqlserver.BindArguments[Config](gqlserver.FieldTypes{
			reflect.TypeOf(iface.LocatorWrapper{}): gqlserver.JSON,
			reflect.TypeOf(tgproducer.Config{}):    tgproducer.GqlConfigInput,
			reflect.TypeOf(fileserver.Config{}):    fileserver.GqlConfigInput,
			reflect.TypeOf(tgconsumer.Config{}):    tgconsumer.GqlConfigInput,
			reflect.TypeOf(fetch.FetcherConfig{}):  fetch.GqlConfigInput,
		}),
		Type: graphql.NewNonNull(GqlTrafficGenType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
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
		Name:        "TgCounters",
		Description: "Traffic generator counters.",
		Fields: gqlserver.BindFields[Counters](gqlserver.FieldTypes{
			reflect.TypeOf(tgproducer.Counters{}): tgproducer.GqlCountersType,
			reflect.TypeOf(fileserver.Counters{}): fileserver.GqlCountersType,
			reflect.TypeOf(tgconsumer.Counters{}): tgconsumer.GqlCountersType,
		}),
	})

	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "Traffic generator counters.",
		Parent:       GqlTrafficGenType.Object,
		Name:         "counters",
		Subscription: "tgCounters",
		FindArgs: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "Traffic generator ID.",
				Type:        gqlserver.NonNullID,
			},
		},
		Find: func(p graphql.ResolveParams) (root any, enders []any, e error) {
			gen := GqlTrafficGenType.Retrieve(p.Args["id"].(string))
			if gen == nil {
				return nil, nil, nil
			}
			return gen, []any{gen.exit}, nil
		},
		Type: GqlCountersType,
		Read: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*TrafficGen).Counters(), nil
		},
	})
}
