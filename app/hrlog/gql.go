package hrlog

import (
	"context"

	"github.com/functionalfoundry/graphqlws"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver/gqlsub"
)

func init() {
	gqlserver.AddSubscription(&graphql.Field{
		Name:        "collectHrlog",
		Description: "Perform hrlog collection.",
		Args: graphql.FieldConfigArgument{
			"filename": &graphql.ArgumentConfig{
				Type: gqlserver.NonNullString,
			},
			"count": &graphql.ArgumentConfig{
				Type: graphql.Int,
			},
		},
		Type: gqlserver.NonNullBoolean,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			if e, ok := p.Info.RootValue.(error); ok {
				return nil, e
			}
			return true, nil
		},
	}, func(ctx context.Context, sub *graphqlws.Subscription, updates chan<- interface{}) {
		defer close(updates)

		if TheWriter == nil {
			updates <- ErrDisabled
			return
		}

		cfg, ok := TaskConfig{}, true
		if cfg.Filename, ok = gqlsub.GetArg(sub, "filename", graphql.String).(string); !ok {
			return
		}
		if cfg.Count, ok = gqlsub.GetArg(sub, "count", graphql.Int).(int); !ok {
			cfg.Count = 0
		}

		updates <- (<-TheWriter.Submit(ctx, cfg))
	})
}
