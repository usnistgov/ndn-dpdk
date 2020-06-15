package running_stat_test

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/running_stat"
)

func TestRunningStat(t *testing.T) {
	assert, _ := makeAR(t)

	a := running_stat.New()
	b := running_stat.New()

	o := a.Read()
	assert.Equal(uint64(0), o.Count())
	assert.Equal(uint64(0), o.Len())
	assert.True(math.IsNaN(o.Min()))
	assert.True(math.IsNaN(o.Max()))
	assert.True(math.IsNaN(o.Mean()))
	assert.True(math.IsNaN(o.Variance()))
	assert.True(math.IsNaN(o.Stdev()))

	// https://en.wikipedia.org/w/index.php?title=Standard_deviation&oldid=821088286
	// "Sample standard deviation of metabolic rate of Northern Fulmars" section "female"
	input := []float64{
		1091.0,
		1490.5,
		1956.1,
		727.7,
		1361.3,
		1086.5,
	}
	for _, x := range input[:3] {
		a.Push(x)
	}
	for _, x := range input[3:] {
		b.Push(x)
	}

	ar := a.Read()
	br := b.Read()

	o = ar.Add(br)
	assert.Equal(uint64(6), o.Count())
	assert.Equal(uint64(6), o.Len())
	assert.InDelta(727.7, o.Min(), 0.1)
	assert.InDelta(1956.1, o.Max(), 0.1)
	assert.InDelta(1285.5, o.Mean(), 0.1)
	assert.InDelta(420.96, o.Stdev(), 0.1)

	// bs := o.Sub(ar)
	bsJson, e := json.Marshal(o.Sub(ar))
	assert.NoError(e)
	var bs running_stat.Snapshot
	e = json.Unmarshal(bsJson, &bs)
	assert.NoError(e)
	assert.Equal(br.Count(), bs.Count())
	assert.Equal(br.Len(), bs.Len())
	assert.True(math.IsNaN(bs.Min()))
	assert.True(math.IsNaN(bs.Max()))
	assert.InDelta(br.Mean(), bs.Mean(), 0.1)
	assert.InDelta(br.Stdev(), bs.Stdev(), 0.1)

	o = o.Scale(10)
	assert.Equal(uint64(6), o.Count())
	assert.Equal(uint64(6), o.Len())
	assert.InDelta(7277, o.Min(), 1.0)
	assert.InDelta(19561, o.Max(), 1.0)
	assert.InDelta(12855, o.Mean(), 1.0)
	assert.InDelta(4209.6, o.Stdev(), 1.0)
}
