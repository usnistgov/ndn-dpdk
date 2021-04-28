package endpoint

import (
	"math"
	"math/rand"
	"time"
)

// RetxIterable is a generator function for successive retransmission intervals.
// Returns zero to disallow further retransmissions.
type RetxIterable func() time.Duration

// RetxPolicy represents an Interest retransmission policy.
type RetxPolicy interface {
	IntervalIterable(lifetime time.Duration) RetxIterable
}

type noRetx struct{}

func (noRetx) IntervalIterable(lifetime time.Duration) RetxIterable {
	return func() time.Duration {
		return 0
	}
}

// RetxOptions specifies how to retransmit an Interest.
type RetxOptions struct {
	// Limit is the maximum number of retransmissions, excluding initial Interest.
	// Default is 0, which disables retransmissions.
	Limit int

	// Interval is the initial retransmission interval.
	// Default is 50% of InterestLifetime.
	Interval time.Duration

	// Randomize causes retransmission interval to be randomized within [1-r, 1+r] range.
	// Suppose this is set to 0.1, an interval of 100ms would become [90ms, 110ms].
	// Default is 0.1. Set a negative value to disable randomization.
	Randomize float64

	// Backoff is the multiplication factor on the interval after each retransmission.
	// Valid range is [1.0, 2.0]. Default is 1.0.
	Backoff float64

	// Max is the maximum retransmission interval.
	// Default is 90% of InterestLifetime.
	Max time.Duration
}

// IntervalIterable implements RetxPolicy.
func (retx RetxOptions) IntervalIterable(lifetime time.Duration) RetxIterable {
	if retx.Interval == 0 {
		retx.Interval = lifetime / 2
	}

	if retx.Randomize == 0 {
		retx.Randomize = 0.1
	} else if retx.Randomize < 0 {
		retx.Randomize = 0
	}

	retx.Backoff = math.Min(math.Max(1.0, retx.Backoff), 2.0)

	if retx.Max == 0 {
		retx.Max = lifetime / 10 * 9
	}
	max := float64(retx.Max)

	count, nextInterval := 0, float64(retx.Interval)
	return func() (d time.Duration) {
		if count >= retx.Limit {
			return 0
		}
		count++

		d = time.Duration(nextInterval * (1 - retx.Randomize + rand.Float64()*2*retx.Randomize))
		nextInterval = math.Min(nextInterval*retx.Backoff, max)
		return d
	}
}
