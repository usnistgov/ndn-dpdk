// Package gqlserver provides a GraphQL server.
// It is a singleton initialized via init() functions.
package gqlserver

import (
	"context"
	"net/http"
	"time"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
	"github.com/sirupsen/logrus"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver/gqlsub"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/mk/version"
	"go.uber.org/zap"
)

var logger = logging.New("gqlserver")

// Schema is the singleton of graphql.SchemaConfig.
// It is available until Prepare() is called.
var Schema = &graphql.SchemaConfig{
	Query: graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: graphql.Fields{},
	}),
	Mutation: graphql.NewObject(graphql.ObjectConfig{
		Name:   "Mutation",
		Fields: graphql.Fields{},
	}),
	Subscription: graphql.NewObject(graphql.ObjectConfig{
		Name:   "Subscription",
		Fields: graphql.Fields{},
	}),
}

// AddQuery adds a top-level query field.
func AddQuery(f *graphql.Field) {
	Schema.Query.AddFieldConfig(f.Name, f)
}

// AddMutation adds a top-level mutation field.
func AddMutation(f *graphql.Field) {
	Schema.Mutation.AddFieldConfig(f.Name, f)
}

// AddSubscription adds a top-level subscription field.
func AddSubscription(f *graphql.Field, h gqlsub.Handler) {
	Schema.Subscription.AddFieldConfig(f.Name, f)
	subHandlers[f.Name] = h
}

func init() {
	AddQuery(&graphql.Field{
		Name: "version",
		Type: graphql.NewNonNull(version.GqlVersionType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return version.Get(), nil
		},
	})

	AddSubscription(&graphql.Field{
		Name:        "tick",
		Description: "time.Ticker subscription for testing subscription implementations.",
		Type:        graphql.NewNonNull(graphql.DateTime),
		Args: graphql.FieldConfigArgument{
			"interval": &graphql.ArgumentConfig{
				Type: graphql.NewNonNull(nnduration.GqlNanoseconds),
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			t := p.Info.RootValue.(time.Time)
			return t, nil
		},
	}, func(ctx context.Context, sub *graphqlws.Subscription, updates chan<- interface{}) {
		defer close(updates)

		interval, ok := gqlsub.GetArg(sub, "interval", nnduration.GqlNanoseconds).(nnduration.Nanoseconds)
		if !ok {
			return
		}

		ticker := time.NewTicker(interval.Duration())
		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				updates <- t
			}
		}
	})
}

var (
	subHandlers = make(gqlsub.HandlerMap)
	subManager  *gqlsub.SubscriptionManager
)

// Prepare compiles the schema and adds handlers on http.DefaultServeMux.
func Prepare() {
	sch, e := graphql.NewSchema(*Schema)
	if e != nil {
		logger.Panic("graphql.NewSchema",
			zap.Error(e),
			zap.Any("schema", Schema),
		)
	}
	Schema = nil

	logrus.SetLevel(logrus.PanicLevel)
	subManager = gqlsub.NewManager(context.Background(), &sch, subHandlers)
	http.Handle("/subscriptions", graphqlws.NewHandler(graphqlws.HandlerConfig{
		SubscriptionManager: subManager,
	}))

	http.Handle("/", handler.New(&handler.Config{
		Schema:           &sch,
		Pretty:           true,
		PlaygroundConfig: handler.NewDefaultPlaygroundConfig(),
	}))
}
