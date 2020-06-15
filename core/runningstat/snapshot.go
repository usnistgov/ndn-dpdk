package runningstat

/*
#include "../../csrc/core/running-stat.h"
*/
import "C"
import (
	"encoding/json"
	"math"
)

// Snapshot contains a snapshot of RunningStat reading.
type Snapshot struct {
	v runningStat
}

// Count returns number of inputs.
func (s Snapshot) Count() uint64 {
	return s.v.I
}

// Len returns number of samples.
func (s Snapshot) Len() uint64 {
	return s.v.N
}

// Min returns minimum value, if enabled.
func (s Snapshot) Min() float64 {
	if s.v.N == 0 {
		return math.NaN()
	}
	return s.v.Min
}

// Max returns maximum value, if enabled.
func (s Snapshot) Max() float64 {
	if s.v.N == 0 {
		return math.NaN()
	}
	return s.v.Max
}

// Means returns mean value.
func (s Snapshot) Mean() float64 {
	if s.v.N == 0 {
		return math.NaN()
	}
	return s.v.M1
}

// Variance returns variance of samples.
func (s Snapshot) Variance() float64 {
	if s.v.N <= 1 {
		return math.NaN()
	}
	return s.v.M2 / float64(s.v.N-1)
}

// Stdev returns standard deviation of samples.
func (s Snapshot) Stdev() float64 {
	return math.Sqrt(s.Variance())
}

// Add combines stats with another instance.
func (a Snapshot) Add(b Snapshot) (c Snapshot) {
	if a.v.I == 0 {
		return b
	} else if b.v.I == 0 {
		return a
	}
	c.v.I = a.v.I + b.v.I
	c.v.N = a.v.N + b.v.N
	c.v.Min = math.Min(a.v.Min, b.v.Min)
	c.v.Max = math.Max(a.v.Max, b.v.Max)
	aN := float64(a.v.N)
	bN := float64(b.v.N)
	cN := float64(c.v.N)
	delta := b.v.M1 - a.v.M1
	delta2 := delta * delta
	c.v.M1 = (aN*a.v.M1 + bN*b.v.M1) / cN
	c.v.M2 = a.v.M2 + b.v.M2 + delta2*aN*bN/cN
	return
}

// Sub subtracts stats in another instance.
func (c Snapshot) Sub(a Snapshot) (b Snapshot) {
	b.v.I = c.v.I - a.v.I
	b.v.N = c.v.N - a.v.N
	b.v.Min = math.NaN()
	b.v.Max = math.NaN()
	cN := float64(c.v.N)
	aN := float64(a.v.N)
	bN := float64(b.v.N)
	b.v.M1 = (cN*c.v.M1 - aN*a.v.M1) / bN
	delta := a.v.M1 - b.v.M1
	delta2 := delta * delta
	b.v.M2 = c.v.M2 - a.v.M2 - delta2*aN*bN/cN
	return b
}

// Scale multiplies every number by a ratio.
func (s Snapshot) Scale(ratio float64) (o Snapshot) {
	o = s
	o.v.Min *= ratio
	o.v.Max *= ratio
	o.v.M1 *= ratio
	o.v.M2 *= ratio * ratio
	return o
}

// MarshalJSON implements json.Marshaler interface.
func (s Snapshot) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["Count"] = s.Count()
	m["Len"] = s.Len()
	m["M1"] = s.v.M1
	m["M2"] = s.v.M2

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

// UnmarshalJSON implements json.Unmarshaler interface.
func (s *Snapshot) UnmarshalJSON(p []byte) (e error) {
	m := make(map[string]interface{})
	if e = json.Unmarshal(p, &m); e != nil {
		return e
	}

	readNum := func(key string) float64 {
		i, ok := m[key]
		if ok {
			v, ok := i.(float64)
			if ok {
				return v
			}
		}
		return math.NaN()
	}
	s.v.I = uint64(readNum("Count"))
	s.v.N = uint64(readNum("Len"))
	s.v.Min = readNum("Min")
	s.v.Max = readNum("Max")
	s.v.M1 = readNum("M1")
	s.v.M2 = readNum("M2")
	return nil
}
