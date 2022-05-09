// Package gqlserver provides a GraphQL server.
// It is a singleton initialized via init() functions.
package gqlserver

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
	goutils "github.com/onichandame/go-utils"
	gqlwsserver "github.com/onichandame/gql-ws/server"
	"github.com/sirupsen/logrus"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/core/version"
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
	f.Resolve = func(p graphql.ResolveParams) (any, error) {
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
		f.Resolve = func(p graphql.ResolveParams) (any, error) {
			return p.Info.RootValue, nil
		}
	}
	if subscribe := f.Subscribe; subscribe != nil {
		f.Subscribe = func(p graphql.ResolveParams) (interface{}, error) {
			var stop chan interface{}
			goutils.Try(func() { stop = gqlwsserver.GetSubscriptionStopSig(p.Context) })
			if stop != nil {
				ctx, cancel := context.WithCancel(p.Context)
				go func() {
					<-stop
					cancel()
				}()
				p.Context = ctx
			}
			return subscribe(p)
		}
	}
	Schema.Subscription.AddFieldConfig(f.Name, f)
}

func init() {
	versionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Version",
		Fields: BindFields[version.Version](FieldTypes{
			reflect.TypeOf(time.Time{}): graphql.DateTime,
		}),
	})
	AddQuery(&graphql.Field{
		Name: "version",
		Type: graphql.NewNonNull(versionType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return version.V, nil
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
	wsHandler := graphqlws.NewHandler(graphqlws.HandlerConfig{
		SubscriptionManager: newSubManager(context.Background(), &sch),
	})
	twsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gqlwsserver.NewSocket(&gqlwsserver.Config{
			Response: w,
			Request:  r,
			Schema:   &sch,
		})
	})
	httpHandler := handler.New(&handler.Config{
		Schema:           &sch,
		Pretty:           true,
		PlaygroundConfig: handler.NewDefaultPlaygroundConfig(),
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("connection") == "Upgrade" {
			if strings.Contains(r.Header.Get("sec-websocket-protocol"), "graphql-transport-ws") {
				twsHandler.ServeHTTP(w, r)
			} else {
				wsHandler.ServeHTTP(w, r)
			}
		} else {
			httpHandler.ServeHTTP(w, r)
		}
	})
}
