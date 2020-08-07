// Package gqlserver provides a GraphQL server.
// It is a singleton and is initialized via init() functions.
package gqlserver

import (
	"net/http"
	"os"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/version"
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
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return version.COMMIT, nil
		},
	})
}

// Start starts the server.
func Start() {
	sch, e := graphql.NewSchema(Schema)
	if e != nil {
		log.WithField("schema", Schema).WithError(e).Panic("graphql.NewSchema")
	}

	go startHTTP(&sch)
}

func startHTTP(sch *graphql.Schema) {
	addr := os.Getenv("GQLSERVER_HTTP")
	switch addr {
	case "0":
		log.Warn("GraphQL HTTP server disabled")
		return
	case "":
		addr = "127.0.0.1:3030"
	}

	h := handler.New(&handler.Config{
		Schema:           sch,
		Pretty:           true,
		PlaygroundConfig: handler.NewDefaultPlaygroundConfig(),
	})
	log.WithField("addr", addr).Info("GraphQL HTTP server starting")

	var mux http.ServeMux
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("User-Agent: *\nDisallow: /\n"))
	})
	mux.Handle("/", h)
	http.ListenAndServe(addr, &mux)
}
