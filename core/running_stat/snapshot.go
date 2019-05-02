package running_stat

import (
	"encoding/json"
	"math"
)

// A snapshot of RunningStat output.
type Snapshot struct {
	Count uint64
	Min   float64
	Max   float64
	Mean  float64
	Stdev float64
}

// Take a snapshot of RunningStat output.
func TakeSnapshot(s RunningStat) (o Snapshot) {
	o.Count = s.Len64()
	o.Min = s.Min()
	o.Max = s.Max()
	o.Mean = s.Mean()
	o.Stdev = s.Stdev()
	return o
}

// Multiple every number by a ratio.
func (s Snapshot) Multiply(ratio float64) (o Snapshot) {
	o = s
	o.Min *= ratio
	o.Max *= ratio
	o.Mean *= ratio
	o.Stdev *= math.Abs(ratio)
	return o
}

func (s Snapshot) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["Count"] = s.Count
	addUnlessNaN := func(key string, value float64) {
		if !math.IsNaN(value) {
			m[key] = value
		}
	}
	addUnlessNaN("Min", s.Min)
	addUnlessNaN("Max", s.Max)
	addUnlessNaN("Mean", s.Mean)
	addUnlessNaN("Stdev", s.Stdev)
	return json.Marshal(m)
}
