package runningstat

import (
	"encoding/json"
	"math"
)

// Snapshot contains a snapshot of RunningStat reading.
type Snapshot struct {
	v runningStat
}

var (
	_ json.Marshaler   = Snapshot{}
	_ json.Unmarshaler = (*Snapshot)(nil)
)

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

// Mean returns mean value.
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

// M1 returns internal variable m1.
func (s Snapshot) M1() float64 {
	return s.v.M1
}

// M2 returns internal variable m1.
func (s Snapshot) M2() float64 {
	return s.v.M2
}

// Add combines stats with another instance.
func (s Snapshot) Add(o Snapshot) (sum Snapshot) {
	if s.v.I == 0 {
		return o
	} else if o.v.I == 0 {
		return s
	}
	sum.v.I = s.v.I + o.v.I
	sum.v.N = s.v.N + o.v.N
	sum.v.Min = math.Min(s.v.Min, o.v.Min)
	sum.v.Max = math.Max(s.v.Max, o.v.Max)
	aN := float64(s.v.N)
	bN := float64(o.v.N)
	cN := float64(sum.v.N)
	delta := o.v.M1 - s.v.M1
	delta2 := delta * delta
	sum.v.M1 = (aN*s.v.M1 + bN*o.v.M1) / cN
	sum.v.M2 = s.v.M2 + o.v.M2 + delta2*aN*bN/cN
	return
}

// Sub computes numerical difference.
func (s Snapshot) Sub(o Snapshot) (diff Snapshot) {
	diff.v.I = s.v.I - o.v.I
	diff.v.N = s.v.N - o.v.N
	diff.v.Min = math.NaN()
	diff.v.Max = math.NaN()
	cN := float64(s.v.N)
	aN := float64(o.v.N)
	bN := float64(diff.v.N)
	diff.v.M1 = (cN*s.v.M1 - aN*o.v.M1) / bN
	delta := o.v.M1 - diff.v.M1
	delta2 := delta * delta
	diff.v.M2 = s.v.M2 - o.v.M2 - delta2*aN*bN/cN
	return diff
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
	m := map[string]interface{}{}
	m["count"] = s.Count()
	m["len"] = s.Len()
	m["m1"] = s.v.M1
	m["m2"] = s.v.M2

	addUnlessNaN := func(key string, value float64) {
		if !math.IsNaN(value) {
			m[key] = value
		}
	}
	addUnlessNaN("min", s.Min())
	addUnlessNaN("max", s.Max())
	addUnlessNaN("mean", s.Mean())
	addUnlessNaN("variance", s.Variance())
	addUnlessNaN("stdev", s.Stdev())
	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (s *Snapshot) UnmarshalJSON(p []byte) (e error) {
	v := struct {
		I   uint64   `json:"count"`
		N   uint64   `json:"len"`
		Min *float64 `json:"min"`
		Max *float64 `json:"max"`
		M1  *float64 `json:"m1"`
		M2  *float64 `json:"m2"`
	}{}
	if e = json.Unmarshal(p, &v); e != nil {
		return e
	}

	readNum := func(x *float64) float64 {
		if x == nil {
			return math.NaN()
		}
		return *x
	}
	s.v.I = v.I
	s.v.N = v.N
	s.v.Min = readNum(v.Min)
	s.v.Max = readNum(v.Max)
	s.v.M1 = readNum(v.M1)
	s.v.M2 = readNum(v.M2)
	return nil
}
