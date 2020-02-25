package running_stat

/*
#include "running-stat.h"
*/
import "C"
import (
	"encoding/json"
	"math"
)

// A snapshot of RunningStat reading.
type Snapshot struct {
	v runningStat
}

// Return number of inputs.
func (s Snapshot) Count() uint64 {
	return s.v.I
}

// Return number of samples.
func (s Snapshot) Len() uint64 {
	return s.v.N
}

// Return minimum value, if enabled.
func (s Snapshot) Min() float64 {
	if s.v.N == 0 {
		return math.NaN()
	}
	return s.v.Min
}

// Return maximum value, if enabled.
func (s Snapshot) Max() float64 {
	if s.v.N == 0 {
		return math.NaN()
	}
	return s.v.Max
}

// Compute mean.
func (s Snapshot) Mean() float64 {
	if s.v.N == 0 {
		return math.NaN()
	}
	return s.v.M1
}

// Compute variance of samples.
func (s Snapshot) Variance() float64 {
	if s.v.N <= 1 {
		return math.NaN()
	}
	return s.v.M2 / float64(s.v.N-1)
}

// Compute standard deviation of samples.
func (s Snapshot) Stdev() float64 {
	return math.Sqrt(s.Variance())
}

// Combine stats with another instance.
func (s *Snapshot) Combine(other Snapshot) {
	s.v.combine(other.v)
}

// Multiple every number by a ratio.
func (s Snapshot) Scale(ratio float64) (o Snapshot) {
	o = s
	o.v.Min *= ratio
	o.v.Max *= ratio
	o.v.M1 *= ratio
	o.v.M2 *= ratio * ratio
	return o
}

func (s Snapshot) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["Count"] = s.Count()
	m["Len"] = s.Len()
	addUnlessNaN := func(key string, value float64) {
		if !math.IsNaN(value) {
			m[key] = value
		}
	}
	addUnlessNaN("Min", s.Min())
	addUnlessNaN("Max", s.Max())
	addUnlessNaN("Mean", s.Mean())
	addUnlessNaN("Variance", s.Variance())
	addUnlessNaN("Stdev", s.Stdev())
	return json.Marshal(m)
}
