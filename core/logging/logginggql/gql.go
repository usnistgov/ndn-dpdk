// Package logginggql allows setting log levels via GraphQL.
package logginggql

import (
	"errors"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/logging"
)

// GraphQL types.
var (
	GqlLoggerType *graphql.Object
)

func init() {
	GqlLoggerType = graphql.NewObject(graphql.ObjectConfig{
		Name: "Logger",
		Fields: graphql.Fields{
			"package": &graphql.Field{
				Description: "Package name.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					pl := p.Source.(logging.PkgLevel)
					return pl.Package(), nil
				},
			},
			"level": &graphql.Field{
				Description: "Log level.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					pl := p.Source.(logging.PkgLevel)
					return string(pl.Level()), nil
				},
			},
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "loggers",
		Description: "Log levels.",
		Type:        gqlserver.NewListNonNullBoth(GqlLoggerType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return logging.ListLevels(), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "setLogLevel",
		Description: "Change log level.",
		Args: graphql.FieldConfigArgument{
			"package": &graphql.ArgumentConfig{
				Description: "Package name.",
				Type:        gqlserver.NonNullString,
			},
			"level": &graphql.ArgumentConfig{
				Description: "Log level.",
				Type:        gqlserver.NonNullString,
			},
		},
		Type: graphql.NewNonNull(GqlLoggerType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			pkg := p.Args["package"].(string)
			lvl := p.Args["level"].(string)
			pl := logging.FindLevel(pkg)
			if pl == nil {
				return nil, errors.New("package not found")
			}
			pl.SetLevel(lvl)
			return *pl, nil
		},
	})
}
