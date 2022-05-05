package runningstat

import (
	"math"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/zyedidia/generic"
)

func combineMinMax(f func(a, b uint64) uint64, a, b *uint64) (uint64, bool) {
	if a == nil || b == nil {
		return 0, false
	}
	return f(*a, *b), true
}

func scaleMinMax(x *uint64, ratio float64) (uint64, bool) {
	if x == nil {
		return 0, false
	}
	return uint64(float64(*x) * ratio), true
}

// Snapshot contains a snapshot of RunningStat reading.
type Snapshot struct {
	Count    uint64  `json:"count" gqldesc:"Number of input values."`
	Len      uint64  `json:"len" gqldesc:"Number of collected samples."`
	Mean     float64 `json:"mean" gqldesc:"Mean value. Valid if count>0."`
	Variance float64 `json:"variance" gqldesc:"Variance of samples. Valid if count>1."`
	Stdev    float64 `json:"stdev" gqldesc:"Standard deviation of samples. Valid if count>1."`
	M1       float64 `json:"m1"`
	M2       float64 `json:"m2"`
	Min      *uint64 `json:"min" gqldesc:"Minimum value. Valid if count>0."`
	Max      *uint64 `json:"max" gqldesc:"Maximum value. Valid if count>0."`
}

// Add combines stats with another instance.
func (s Snapshot) Add(o Snapshot) Snapshot {
	if s.Count == 0 {
		return o
	} else if o.Count == 0 {
		return s
	}
	i := s.Count + o.Count
	n := s.Len + o.Len
	aN, bN, cN := float64(s.Len), float64(o.Len), float64(n)
	delta := o.M1 - s.M1
	delta2 := delta * delta
	m1 := (aN*s.M1 + bN*o.M1) / cN
	m2 := s.M2 + o.M2 + delta2*aN*bN/cN
	min, hasMin := combineMinMax(generic.Min[uint64], s.Min, o.Min)
	max, hasMax := combineMinMax(generic.Max[uint64], s.Max, o.Max)
	return newSnapshot(i, n, m1, m2, hasMin && hasMax, min, max)
}

// Sub computes numerical difference.
func (s Snapshot) Sub(o Snapshot) Snapshot {
	i := s.Count - o.Count
	n := s.Len - o.Len
	cN, aN, bN := float64(s.Len), float64(o.Len), float64(n)
	m1 := (cN*s.M1 - aN*o.M1) / bN
	delta := o.M1 - m1
	delta2 := delta * delta
	m2 := s.M2 - o.M2 - delta2*aN*bN/cN
	return newSnapshot(i, n, m1, m2, false, 0, 0)
}

// Scale multiplies every number by a ratio.
func (s Snapshot) Scale(ratio float64) Snapshot {
	m1, m2 := s.M1*ratio, s.M2*ratio*ratio
	min, hasMin := scaleMinMax(s.Min, ratio)
	max, hasMax := scaleMinMax(s.Max, ratio)
	return newSnapshot(s.Count, s.Len, m1, m2, hasMin && hasMax, min, max)
}

func newSnapshot(i, n uint64, m1, m2 float64, hasMinMax bool, min, max uint64) (s Snapshot) {
	s.Count, s.Len = i, n
	s.M1, s.M2 = m1, m2
	if n > 0 {
		s.Mean = m1
	}
	if n > 1 {
		s.Variance = m2 / float64(n-1)
		s.Stdev = math.Sqrt(s.Variance)
	}
	if n > 0 && hasMinMax {
		s.Min, s.Max = &min, &max
	}
	return
}

// GqlSnapshotType is the GraphQL type for Snapshot.
var GqlSnapshotType = graphql.NewObject(graphql.ObjectConfig{
	Name:   "RunningStatSnapshot",
	Fields: gqlserver.BindFields[Snapshot](nil),
})
