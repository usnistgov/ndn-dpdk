package gqlserver

import (
	"reflect"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/core/subtract"
	"golang.org/x/exp/maps"
)

// PublishChan publishes a channel in reply to GraphQL subscription.
//
// f is a callback function that sends its results into a channel.
// It should not close the channel - the channel will be closed by the caller when f returns.
func PublishChan(f func(updates chan<- any)) (any, error) {
	updates := make(chan any)
	go func() {
		defer close(updates)
		f(updates)
	}()
	return updates, nil
}

var (
	subArgInterval = graphql.FieldConfigArgument{
		"interval": &graphql.ArgumentConfig{
			Description:  "Interval between updates.",
			Type:         nnduration.GqlNanoseconds,
			DefaultValue: nnduration.Nanoseconds(time.Second),
		},
	}
	subArgDiff = graphql.FieldConfigArgument{
		"diff": &graphql.ArgumentConfig{
			Description:  "Report value difference since last update instead of accumulative total.",
			Type:         graphql.Boolean,
			DefaultValue: false,
		},
	}
)

// PublishInterval publishes results at an interval in reply to GraphQL subscription.
//
// read is a callback function that returns a single result.
// enders are channels that indicate the subscription should be canceled, when a value is received or the channel is closed.
func PublishInterval(p graphql.ResolveParams, read graphql.FieldResolveFn, enders ...any) (results any, e error) {
	interval := p.Args["interval"].(nnduration.Nanoseconds).Duration()

	diff := false
	var prev any
	if diffB, ok := p.Args["diff"].(bool); ok && diffB {
		diff = true
		if prev, e = read(p); e != nil || prev == nil {
			return nil, e
		}
	}

	return PublishChan(func(updates chan<- any) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		cont := true
		doUpdate := func() {
			value, e := read(p)
			switch {
			case e != nil, value == nil:
				cont = false
			case diff:
				updates <- subtract.Sub(value, prev)
				prev = value
			case !diff:
				updates <- value
			}
		}

		if len(enders) == 0 {
			for cont {
				select {
				case <-ticker.C:
					doUpdate()
				case <-p.Context.Done():
					return
				}
			}
		}

		cases := []reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ticker.C)},
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(p.Context.Done())},
		}
		for _, ender := range enders {
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ender)})
		}
		for cont {
			if i, _, _ := reflect.Select(cases); i != 0 {
				return
			}
			doUpdate()
		}
	})
}

func init() {
	AddSubscription(&graphql.Field{
		Name:        "tick",
		Description: "time.Ticker subscription for testing subscription implementations.",
		Type:        NonNullInt,
		Args:        subArgInterval,
		Subscribe: func(p graphql.ResolveParams) (any, error) {
			n := 0
			return PublishInterval(p, func(graphql.ResolveParams) (any, error) {
				n++
				return n, nil
			})
		},
	})
}

// CountersConfig contains options to publish a counters-like type.
type CountersConfig struct {
	Description string

	// Parent is the parent object to define a query field.
	// If nil, no query field is defined.
	Parent *graphql.Object
	// Name is the field name to be defined in Parent.
	// If empty, no query field is defined.
	Name string

	// Subscription is the field name to be defined as subscription.
	// If empty, no subscription is defined.
	Subscription string
	// NoDiff indicates diff=true is not supported.
	// This should be set if some fields are lazily-injected with FieldResolveFn, as subtract algorithm cannot see them.
	NoDiff bool
	// FindArgs declares GraphQL arguments for finding the source object.
	FindArgs graphql.FieldConfigArgument
	// Find finds source object from FindArgs in a subscription.
	// p.Source is unspecified.
	// p.Args contains arguments declared in both Args and FindArgs.
	// enders: see PublishInterval.
	Find func(p graphql.ResolveParams) (source any, enders []any, e error)

	// Type declares GraphQL return type.
	Type graphql.Output
	// Args declares GraphQL arguments for customizing counters.
	Args graphql.FieldConfigArgument
	// Read retrieves counters from p.Source.
	// p.Source is a value from Parent or a return value from Find.
	// p.Args contains arguments declared in Args.
	Read graphql.FieldResolveFn
}

func (cfg CountersConfig) subscribe(p graphql.ResolveParams) (any, error) {
	source, enders, e := cfg.Find(p)
	if e != nil {
		return nil, e
	}
	if val := reflect.ValueOf(source); !val.IsValid() || val.IsZero() {
		return nil, nil
	}

	p.Source = source
	return PublishInterval(p, cfg.Read, enders...)
}

// AddCounters defines a counters-like field as both a query field and a subscription.
func AddCounters(cfg *CountersConfig) {
	if cfg.Parent != nil && cfg.Name != "" {
		cfg.Parent.AddFieldConfig(cfg.Name, &graphql.Field{
			Description: cfg.Description,
			Args:        cfg.Args,
			Type:        cfg.Type,
			Resolve:     cfg.Read,
		})
	}

	if cfg.Subscription != "" {
		args := graphql.FieldConfigArgument{}
		maps.Copy(args, cfg.Args)
		maps.Copy(args, cfg.FindArgs)
		maps.Copy(args, subArgInterval)
		if !cfg.NoDiff {
			maps.Copy(args, subArgDiff)
		}
		AddSubscription(&graphql.Field{
			Name:        cfg.Subscription,
			Description: cfg.Description,
			Args:        args,
			Type:        cfg.Type,
			Subscribe:   cfg.subscribe,
		})
	}
}
