package fetch

import (
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// GqlRetrieveByFaceID returns *Fetcher associated with a face.
// It is assigned during package tg initialization.
var GqlRetrieveByFaceID func(id iface.ID) *Fetcher

// GraphQL types.
var (
	GqlConfigInput     *graphql.InputObject
	GqlTaskDefInput    *graphql.InputObject
	GqlTaskDefType     *graphql.Object
	GqlTaskContextType *gqlserver.NodeType[*TaskContext]
	GqlFetcherType     *gqlserver.NodeType[*Fetcher]
)

func init() {
	GqlConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FetcherConfigInput",
		Description: "Fetcher config.",
		Fields: gqlserver.BindInputFields[Config](gqlserver.FieldTypes{
			reflect.TypeOf(iface.PktQueueConfig{}): iface.GqlPktQueueInput,
		}),
	})

	GqlTaskDefInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FetchTaskDefInput",
		Description: "Fetch task definition.",
		Fields:      gqlserver.BindInputFields[TaskDef](ndni.GqlInterestTemplateFieldTypes),
	})
	GqlTaskDefType = graphql.NewObject(graphql.ObjectConfig{
		Name:        "FetchTaskDef",
		Description: "Fetch task definition.",
		Fields:      gqlserver.BindFields[TaskDef](ndni.GqlInterestTemplateFieldTypes),
	})

	GqlTaskContextType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name:        "FetchTaskContext",
		Description: "Fetch task context.",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Type: gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					task := p.Source.(*TaskContext)
					return task.id, nil
				},
			},
			"task": &graphql.Field{
				Description: "Task definition.",
				Type:        graphql.NewNonNull(GqlTaskDefType),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					task := p.Source.(*TaskContext)
					return task.d, nil
				},
			},
			"worker": ealthread.GqlWithWorker(func(p graphql.ResolveParams) ealthread.Thread {
				task := p.Source.(*TaskContext)
				return task.w
			}),
		},
	}, gqlserver.NodeConfig[*TaskContext]{
		RetrieveInt: func(id int) *TaskContext {
			taskContextLock.RLock()
			defer taskContextLock.RUnlock()
			return taskContextByID[id]
		},
		Delete: func(task *TaskContext) error {
			task.Stop()
			return nil
		},
	})

	GqlFetcherType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "Fetcher",
		Fields: tggql.CommonFields(graphql.Fields{
			"tasks": &graphql.Field{
				Description: "Running tasks.",
				Type:        gqlserver.NewListNonNullBoth(GqlTaskContextType.Object),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					fetcher := p.Source.(*Fetcher)
					return fetcher.Tasks(), nil
				},
			},
		}),
	}, tggql.NodeConfig(&GqlRetrieveByFaceID))

	gqlserver.AddMutation(&graphql.Field{
		Name:        "fetch",
		Description: "Start a fetch task.",
		Args: graphql.FieldConfigArgument{
			"fetcher": &graphql.ArgumentConfig{
				Description: "Fetcher ID.",
				Type:        gqlserver.NonNullID,
			},
			"task": &graphql.ArgumentConfig{
				Description: "Task definition.",
				Type:        graphql.NewNonNull(GqlTaskDefInput),
			},
		},
		Type: graphql.NewNonNull(GqlTaskContextType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			fetcher := GqlFetcherType.Retrieve(p.Args["fetcher"].(string))
			if fetcher == nil {
				return nil, errors.New("fetcher not found")
			}

			var d TaskDef
			if e := jsonhelper.Roundtrip(p.Args["task"], &d, jsonhelper.DisallowUnknownFields); e != nil {
				return nil, e
			}

			return fetcher.Fetch(d)
		},
	})

	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "Fetch task progress and congestion control counters.",
		Parent:       GqlTaskContextType.Object,
		Name:         "counters",
		Subscription: "fetchCounters",
		NoDiff:       true,
		FindArgs: graphql.FieldConfigArgument{
			"task": &graphql.ArgumentConfig{
				Description: "Task context.",
				Type:        gqlserver.NonNullID,
			},
		},
		Find: func(p graphql.ResolveParams) (source any, enders []any, e error) {
			task := GqlTaskContextType.Retrieve(p.Args["task"].(string))
			if task == nil {
				return nil, nil, nil
			}
			return task, []any{task.stopping}, nil
		},
		Type: gqlserver.NonNullJSON,
		Read: func(p graphql.ResolveParams) (any, error) {
			task := p.Source.(*TaskContext)
			return task.Counters(), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:              "runFetchBenchmark",
		Description:       "Run a fetcher benchmark.",
		DeprecationReason: "Use fetch mutation and fetchCounters subscription.",
		Args: graphql.FieldConfigArgument{
			"fetcher": &graphql.ArgumentConfig{
				Description: "Fetcher ID.",
				Type:        gqlserver.NonNullID,
			},
			"templates": &graphql.ArgumentConfig{
				Description: "Interest templates.",
				Type:        gqlserver.NewListNonNullBoth(ndni.GqlInterestTemplateInput),
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
		Resolve: func(p graphql.ResolveParams) (any, error) {
			fetcher := GqlFetcherType.Retrieve(p.Args["fetcher"].(string))
			if fetcher == nil {
				return nil, errors.New("fetcher not found")
			}

			var templates []ndni.InterestTemplateConfig
			if e := jsonhelper.Roundtrip(p.Args["templates"], &templates, jsonhelper.DisallowUnknownFields); e != nil {
				return nil, e
			}
			finalSegNums, _ := p.Args["finalSegNum"].([]any)

			fetcher.Reset()
			var tasks []*TaskContext
			for i, tpl := range templates {
				d := TaskDef{
					InterestTemplateConfig: tpl,
				}
				if i < len(finalSegNums) {
					d.SegmentEnd = uint64(finalSegNums[i].(int)) + 1
				}

				task, e := fetcher.Fetch(d)
				if e != nil {
					//lint:ignore ST1005 'Fetch' is a function name
					return nil, fmt.Errorf("Fetch[%d]: %w", i, e)
				}
				tasks = append(tasks, task)
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
				for i, task := range tasks {
					result[i][c] = task.Counters()
				}
			}
			fetcher.Stop()
			return result, nil
		},
	})
}
