package fetch

import (
	"fmt"
	"reflect"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// GqlRetrieveByFaceID returns *Fetcher associated with a face.
// It is assigned during package tg initialization.
var GqlRetrieveByFaceID func(id iface.ID) interface{}

// GraphQL types.
var (
	GqlConfigInput     *graphql.InputObject
	GqlTemplateInput   *graphql.InputObject
	GqlFetcherNodeType *gqlserver.NodeType
	GqlFetcherType     *graphql.Object
)

type benchmarkTemplate struct {
	Prefix           ndn.Name                `json:"prefix"`
	InterestLifetime nnduration.Milliseconds `json:"interestLifetime,omitempty"`
	CanBePrefix      bool                    `json:"canBePrefix,omitempty"`
}

func init() {
	GqlConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FetcherConfigInput",
		Description: "Fetcher config.",
		Fields: gqlserver.BindInputFields(FetcherConfig{}, gqlserver.FieldTypes{
			reflect.TypeOf(iface.PktQueueConfig{}): iface.GqlPktQueueInput,
		}),
	})

	GqlTemplateInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FetchTemplateInput",
		Description: "Fetcher template.",
		Fields: gqlserver.BindInputFields(benchmarkTemplate{}, gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}):                 gqlserver.NonNullString,
			reflect.TypeOf(nnduration.Milliseconds(0)): nnduration.GqlMilliseconds,
		}),
	})

	GqlFetcherNodeType = tggql.NewNodeType((*Fetcher)(nil), &GqlRetrieveByFaceID)
	GqlFetcherType = graphql.NewObject(GqlFetcherNodeType.Annotate(graphql.ObjectConfig{
		Name:   "Fetcher",
		Fields: tggql.CommonFields(graphql.Fields{}),
	}))
	GqlFetcherNodeType.Register(GqlFetcherType)

	gqlserver.AddMutation(&graphql.Field{
		Name:        "runFetchBenchmark",
		Description: "Execute a fetcher benchmark.",
		Args: graphql.FieldConfigArgument{
			"fetcher": &graphql.ArgumentConfig{
				Description: "Fetcher ID.",
				Type:        gqlserver.NonNullID,
			},
			"templates": &graphql.ArgumentConfig{
				Description: "Interest templates.",
				Type:        gqlserver.NewNonNullList(GqlTemplateInput),
			},
			"interval": &graphql.ArgumentConfig{
				Description: "How often to collect statistics.",
				Type:        graphql.NewNonNull(nnduration.GqlNanoseconds),
			},
			"count": &graphql.ArgumentConfig{
				Description: "How many sets of statistics to be collected.",
				Type:        gqlserver.NonNullInt,
			},
		},
		Type: gqlserver.NonNullJSON,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var fetcher *Fetcher
			if e := gqlserver.RetrieveNodeOfType(GqlFetcherNodeType, p.Args["fetcher"], &fetcher); e != nil {
				return nil, e
			}

			var templates []benchmarkTemplate
			if e := jsonhelper.Roundtrip(p.Args["templates"], &templates, jsonhelper.DisallowUnknownFields); e != nil {
				return nil, e
			}

			fetcher.Reset()
			var logics []*Logic
			for i, tpl := range templates {
				tplArgs := []interface{}{tpl.Prefix}
				if tpl.CanBePrefix {
					tplArgs = append(tplArgs, ndn.CanBePrefixFlag)
				}
				if d := tpl.InterestLifetime.Duration(); d > 0 {
					tplArgs = append(tplArgs, d)
				}
				if _, e := fetcher.AddTemplate(tplArgs...); e != nil {
					return nil, fmt.Errorf("AddTemplate[%d]: %w", i, e)
				}
				logics = append(logics, fetcher.Logic(i))
			}

			interval := p.Args["interval"].(nnduration.Nanoseconds)
			count := p.Args["count"].(int)

			result := make([][]Counters, len(templates))
			for i := range result {
				result[i] = make([]Counters, count)
			}

			fetcher.Launch()
			ticker := time.NewTicker(interval.Duration())
			defer ticker.Stop()
			for c := 0; c < count; c++ {
				<-ticker.C
				for i, logic := range logics {
					result[i][c] = logic.Counters()
				}
			}
			fetcher.Stop()
			return result, nil
		},
	})
}
