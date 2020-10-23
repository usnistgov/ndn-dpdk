// Package gqlserver provides a GraphQL server.
// It is a singleton and is initialized via init() functions.
package gqlserver

import (
	"net/http"
	"net/url"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/mk/version"
)

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
		log.WithField("schema", Schema).WithError(e).Panic("graphql.NewSchema")
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
		log.WithError(e).Warn("gqlserver URI invalid, using the default")
		return
	}

	if u.Scheme != "http" {
		log.Warn("gqlserver URI is not HTTP, using the default")
		return
	}

	if u.User != nil || u.Path != "/" || len(u.Query()) > 0 {
		log.Warn("gqlserver URI contains User/Path/Query, ignored")
	}
	return u.Host
}

func startHTTP(sch *graphql.Schema, listen string) {
	h := handler.New(&handler.Config{
		Schema:           sch,
		Pretty:           true,
		PlaygroundConfig: handler.NewDefaultPlaygroundConfig(),
	})
	log.WithField("listen", listen).Info("GraphQL HTTP server starting")

	var mux http.ServeMux
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("User-Agent: *\nDisallow: /\n"))
	})
	mux.Handle("/", h)
	http.ListenAndServe(listen, &mux)
}
