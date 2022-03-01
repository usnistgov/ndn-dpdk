// Package gqlserver provides a GraphQL server.
// It is a singleton initialized via init() functions.
package gqlserver

import (
	"context"
	"net/http"
	"reflect"
	"time"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
	"github.com/sirupsen/logrus"
	"github.com/usnistgov/ndn-dpdk/core/logging"
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
	name, resolve := f.Name, f.Resolve
	f.Resolve = func(p graphql.ResolveParams) (interface{}, error) {
		defer func() {
			if e := recover(); e != nil {
				logger.Error("panic in GraphQL mutation resolver",
					zap.String("name", name),
					zap.Any("error", e),
					zap.StackSkip("stack", 2),
				)
				panic(e)
			}
		}()
		return resolve(p)
	}
	Schema.Mutation.AddFieldConfig(f.Name, f)
}

// AddSubscription adds a top-level subscription field.
func AddSubscription(f *graphql.Field) {
	if f.Resolve == nil {
		f.Resolve = func(p graphql.ResolveParams) (interface{}, error) {
			return p.Info.RootValue, nil
		}
	}
	Schema.Subscription.AddFieldConfig(f.Name, f)
}

func init() {
	versionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Version",
		Fields: BindFields(version.Version{}, FieldTypes{
			reflect.TypeOf(time.Time{}): graphql.DateTime,
		}),
	})
	AddQuery(&graphql.Field{
		Name: "version",
		Type: graphql.NewNonNull(versionType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return version.Get(), nil
		},
	})
}

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
	http.Handle("/subscriptions", graphqlws.NewHandler(graphqlws.HandlerConfig{
		SubscriptionManager: newSubManager(context.Background(), &sch),
	}))

	http.Handle("/", handler.New(&handler.Config{
		Schema:           &sch,
		Pretty:           true,
		PlaygroundConfig: handler.NewDefaultPlaygroundConfig(),
	}))
}
