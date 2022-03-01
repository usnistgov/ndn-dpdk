package gqlserver

import (
	"reflect"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
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

// IntervalArgs adds 'interval' and 'diff' arguments.
func IntervalArgs(value interface{}, args graphql.FieldConfigArgument) graphql.FieldConfigArgument {
	if args == nil {
		args = graphql.FieldConfigArgument{}
	}

	args["interval"] = &graphql.ArgumentConfig{
		Description:  "Interval between updates.",
		Type:         nnduration.GqlNanoseconds,
		DefaultValue: nnduration.Nanoseconds(time.Second),
	}

	typ := reflect.TypeOf(value)
	sub, hasSub := typ.MethodByName("Sub")
	if hasSub && sub.Type.NumIn() == 2 && sub.Type.In(1) == typ && sub.Type.NumOut() == 1 && sub.Type.Out(0) == typ {
		args["diff"] = &graphql.ArgumentConfig{
			Description: "Report value difference since last update instead of accumulative total.",
			Type:        graphql.Boolean,
		}
	}

	return args
}

// PublishChan publishes results at an interval in reply to GraphQL subscription.
//
// read is a callback function that retrieves a single result.
// enders are channels that indicate the subscription should be canceled, when a value is received or the channel is closed.
func PublishInterval(p graphql.ResolveParams, read func() interface{}, enders ...interface{}) (interface{}, error) {
	interval := p.Args["interval"].(nnduration.Nanoseconds).Duration()

	var prev reflect.Value
	var sub reflect.Value
	diff, _ := p.Args["diff"].(bool)
	if diff {
		prev = reflect.ValueOf(read())
		subMethod, _ := prev.Type().MethodByName("Sub")
		sub = subMethod.Func
	}

	return PublishChan(func(updates chan<- interface{}) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		cases := []reflect.SelectCase{
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ticker.C)},
			{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(p.Context.Done())},
		}
		for _, ender := range enders {
			cases = append(cases, reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ender)})
		}

		for {
			i, _, _ := reflect.Select(cases)
			if i != 0 {
				break
			}

			value := read()
			if diff {
				val := reflect.ValueOf(value)
				delta := sub.Call([]reflect.Value{val, prev})
				updates <- delta[0].Interface()
				prev = val
			} else {
				updates <- value
			}
		}
	})
}

func init() {
	AddSubscription(&graphql.Field{
		Name:        "tick",
		Description: "time.Ticker subscription for testing subscription implementations.",
		Type:        NonNullInt,
		Args:        IntervalArgs(int(0), nil),
		Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
			n := 0
			return PublishInterval(p, func() interface{} {
				n++
				return n
			})
		},
	})
}
