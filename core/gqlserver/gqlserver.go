package gqlserver

import (
	"net/http"
	"os"

	"github.com/bhoriuchi/graphql-go-tools/handler"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/version"
)

// Schema is the singleton of graphql.SchemaConfig.
var Schema graphql.SchemaConfig

// AddQuery adds a top-level query field.
func AddQuery(f *graphql.Field) {
	Schema.Query.AddFieldConfig(f.Name, f)
}

func init() {
	Schema = graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"version": &graphql.Field{
					Type: graphql.String,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return version.COMMIT, nil
					},
				},
			},
		}),
	}
}

// Start starts the server.
func Start() {
	sch, e := graphql.NewSchema(Schema)
	if e != nil {
		log.WithField("schema", Schema).WithError(e).Panic("graphql.NewSchema")
	}

	addr := os.Getenv("GQLSERVER")
	switch addr {
	case "0":
		log.Warn("GraphQL server disabled")
		return
	case "":
		addr = "127.0.0.1:3030"
	}

	h := handler.New(&handler.Config{
		Schema:           &sch,
		Pretty:           true,
		PlaygroundConfig: handler.NewDefaultPlaygroundConfig(),
	})

	var mux http.ServeMux
	mux.HandleFunc("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain")
		w.Write([]byte("User-Agent: *\nDisallow: /\n"))
	})
	mux.Handle("/", h)
	go http.ListenAndServe(addr, &mux)

	log.WithField("addr", addr).Info("GraphQL server started")
}
