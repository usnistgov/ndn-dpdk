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
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// GqlRetrieveByFaceID returns *Fetcher associated with a face.
// It is assigned during package tg initialization.
var GqlRetrieveByFaceID func(id iface.ID) interface{}

// GraphQL types.
var (
	GqlConfigInput     *graphql.InputObject
	GqlFetcherNodeType *gqlserver.NodeType
	GqlFetcherType     *graphql.Object
)

func init() {
	GqlConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FetcherConfigInput",
		Description: "Fetcher config.",
		Fields: gqlserver.BindInputFields(FetcherConfig{}, gqlserver.FieldTypes{
			reflect.TypeOf(iface.PktQueueConfig{}): iface.GqlPktQueueInput,
		}),
	})

	GqlFetcherNodeType = tggql.NewNodeType("Fetcher", (*Fetcher)(nil), &GqlRetrieveByFaceID)
	GqlFetcherType = graphql.NewObject(GqlFetcherNodeType.Annotate(graphql.ObjectConfig{
		Name:   "Fetcher",
		Fields: tggql.CommonFields(graphql.Fields{}),
	}))
	GqlFetcherNodeType.Register(GqlFetcherType)

	gqlserver.AddMutation(&graphql.Field{
		Name:        "runFetchBenchmark",
		Description: "Run a fetcher benchmark.",
		Args: graphql.FieldConfigArgument{
			"fetcher": &graphql.ArgumentConfig{
				Description: "Fetcher ID.",
				Type:        gqlserver.NonNullID,
			},
			"templates": &graphql.ArgumentConfig{
				Description: "Interest templates.",
				Type:        gqlserver.NewNonNullList(ndni.GqlInterestTemplateInput),
			},
			"finalSegNum": &graphql.ArgumentConfig{
				Description: "Final segment number for each Interest template",
				Type:        graphql.NewList(gqlserver.NonNullInt),
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

			var templates []ndni.InterestTemplateConfig
			if e := jsonhelper.Roundtrip(p.Args["templates"], &templates, jsonhelper.DisallowUnknownFields); e != nil {
				return nil, e
			}
			finalSegNums, _ := p.Args["finalSegNum"].([]interface{})

			fetcher.Reset()
			var logics []*Logic
			for _, tpl := range templates {
				i, e := fetcher.AddTemplate(tpl)
				if e != nil {
					return nil, fmt.Errorf("AddTemplate[%d]: %w", i, e)
				}
				logic := fetcher.Logic(i)
				if i < len(finalSegNums) {
					logic.SetFinalSegNum(uint64(finalSegNums[i].(int)))
				}
				logics = append(logics, logic)
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
