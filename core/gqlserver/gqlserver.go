// Package gqlserver provides a GraphQL server.
// It is a singleton and is initialized via init() functions.
package gqlserver

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/mk/version"
	"go.uber.org/zap"
)

var logger = logging.New("gqlserver")

// Schema is the singleton of graphql.SchemaConfig.
var Schema = graphql.SchemaConfig{
	Query: graphql.NewObject(graphql.ObjectConfig{
		Name:   "Query",
		Fields: graphql.Fields{},
	}),
	Mutation: graphql.NewObject(graphql.ObjectConfig{
		Name:   "Mutation",
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

func init() {
	AddQuery(&graphql.Field{
		Name: "version",
		Type: graphql.NewNonNull(version.GqlVersionType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return version.Get(), nil
		},
	})
}

// Start starts the server.
func Start(uri string) {
	sch, e := graphql.NewSchema(Schema)
	if e != nil {
		logger.Panic("graphql.NewSchema",
			zap.Error(e),
			zap.Any("schema", Schema),
		)
	}

	go startHTTP(&sch, parseListenAddress(uri))
}

func parseListenAddress(uri string) (listen string) {
	listen = "127.0.0.1:3030"
	if uri == "" {
		return
	}

	u, e := url.Parse(uri)
	if e != nil {
		logger.Warn("gqlserver URI invalid, using the default", zap.Error(e))
		return
	}

	if u.Scheme != "http" {
		logger.Warn("gqlserver URI is not HTTP, using the default")
		return
	}

	if u.User != nil || strings.TrimPrefix(u.Path, "/") != "" || u.RawQuery != "" {
		logger.Warn("gqlserver URI contains User/Path/Query, ignored")
	}
	return u.Host
}

func startHTTP(sch *graphql.Schema, listen string) {
	h := handler.New(&handler.Config{
		Schema:           sch,
		Pretty:           true,
		PlaygroundConfig: handler.NewDefaultPlaygroundConfig(),
	})
	logger.Info("GraphQL HTTP server starting",
		zap.String("listen", listen),
	)

	var mux http.ServeMux
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("User-Agent: *\nDisallow: /\n"))
	})
	mux.Handle("/", h)
	http.ListenAndServe(listen, &mux)
}
