package gqlserver

import (
	"reflect"
	"time"

	"github.com/VojtechVitek/mergemaps"
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/core/subtract"
)

// PublishChan publishes a channel in reply to GraphQL subscription.
//
// f is a callback function that sends its results into a channel.
// It should not close the channel - the channel will be closed by the caller when f returns.
func PublishChan(f func(updates chan<- interface{})) (interface{}, error) {
	updates := make(chan interface{})
	go func() {
		defer close(updates)
		f(updates)
	}()
	return updates, nil
}

// IntervalDiffArgs adds 'interval' and 'diff' arguments.
func IntervalDiffArgs(args graphql.FieldConfigArgument) graphql.FieldConfigArgument {
	if args == nil {
		args = graphql.FieldConfigArgument{}
	}

	args["interval"] = &graphql.ArgumentConfig{
		Description:  "Interval between updates.",
		Type:         nnduration.GqlNanoseconds,
		DefaultValue: nnduration.Nanoseconds(time.Second),
	}

	args["diff"] = &graphql.ArgumentConfig{
		Description:  "Report value difference since last update instead of accumulative total.",
		Type:         graphql.Boolean,
		DefaultValue: false,
	}

	return args
}

// Publish publishes results at an interval in reply to GraphQL subscription.
//
// read is a callback function that returns a single result.
// enders are channels that indicate the subscription should be canceled, when a value is received or the channel is closed.
func PublishInterval(p graphql.ResolveParams, read graphql.FieldResolveFn, enders ...interface{}) (results interface{}, e error) {
	interval := p.Args["interval"].(nnduration.Nanoseconds).Duration()

	diff := false
	var prev interface{}
	if diff = p.Args["diff"].(bool); diff {
		if prev, e = read(p); e != nil {
			return nil, e
		}
	}

	return PublishChan(func(updates chan<- interface{}) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		cont := true
		doUpdate := func() {
			value, e := read(p)
			if e != nil {
				cont = false
				return
			}
			if diff {
				updates <- subtract.Sub(value, prev)
				prev = value
			} else {
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
		Args:        IntervalDiffArgs(nil),
		Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
			n := 0
			return PublishInterval(p, func(p graphql.ResolveParams) (interface{}, error) {
				n++
				return n, nil
			})
		},
	})
}

// Counters contains options to publish a counters-like type.
type Counters struct {
	Description string
	// Args declares GraphQL arguments for customizing counters.
	Args graphql.FieldConfigArgument
	// Type declares GraphQL return type.
	Type graphql.Output

	// Parent is the parent object to define a query field.
	// If nil, no query field is defined.
	Parent *graphql.Object
	// Name is the field name to be defined in Parent.
	// If empty, no query field is defined.
	Name string

	// Subscription is the field name to be defined as subscription.
	// If empty, no subscription is defined.
	Subscription string
	// FindArgs declares GraphQL arguments for finding the source object.
	FindArgs graphql.FieldConfigArgument
	// Find finds source object and channels that would cancel the subscription.
	// p.Source is unspecified.
	// p.Args contains arguments declared in both Args and FindArgs.
	// This is only invoked for subscription operation.
	Find func(p graphql.ResolveParams) (source interface{}, enders []interface{}, e error)

	// Read retrieves counters from p.Source.
	//  p.Source is a value from Parent or a return value from Find.
	//  p.Args contains arguments declared in Args.
	Read graphql.FieldResolveFn
}

func (cfg Counters) subscribe(p graphql.ResolveParams) (interface{}, error) {
	source, enders, e := cfg.Find(p)
	if e != nil {
		return nil, e
	}

	p.Source = source
	return PublishInterval(p, cfg.Read, enders...)
}

// AddCounters defines a counters-like field as both a query field and a subscription.
func AddCounters(cfg *Counters) {
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
		mergemaps.MergeInto(args, cfg.Args, 0)
		mergemaps.MergeInto(args, cfg.FindArgs, 0)
		AddSubscription(&graphql.Field{
			Name:      cfg.Subscription,
			Args:      IntervalDiffArgs(args),
			Type:      cfg.Type,
			Subscribe: cfg.subscribe,
		})
	}
}
