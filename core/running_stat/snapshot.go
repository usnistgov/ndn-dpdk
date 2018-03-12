package running_stat

import "math"

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
	o.Count = s.Count
	o.Min = s.Min * ratio
	o.Max = s.Max * ratio
	o.Mean = s.Mean * ratio
	o.Stdev = s.Stdev * math.Abs(ratio)
	return o
}
