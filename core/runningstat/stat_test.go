package runningstat_test

import (
	"encoding/json"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var makeAR = testenv.MakeAR

// https://en.wikipedia.org/w/index.php?title=Standard_deviation&oldid=821088286
// "Sample standard deviation of metabolic rate of Northern Fulmars" section "female"
var (
	input = []float64{
		1091.0,
		1490.5,
		1956.1,
		727.7,
		1361.3,
		1086.5,
	}
	min   = 727.7
	max   = 1956.1
	mean  = 1285.5
	stdev = 420.96
)

func TestRunningStat(t *testing.T) {
	assert, _ := makeAR(t)

	var a, b runningstat.RunningStat
	a.Init(1)
	b.Init(1)

	o := a.Read()
	assert.EqualValues(0, o.Count)
	assert.EqualValues(0, o.Len)
	assert.Nil(o.Min)
	assert.Nil(o.Max)

	for _, x := range input[:3] {
		a.Push(x)
	}
	for _, x := range input[3:] {
		b.Push(x)
	}

	ar, br := a.Read(), b.Read()

	o = ar.Add(br)
	assert.EqualValues(6, o.Count)
	assert.EqualValues(6, o.Len)
	assert.Nil(o.Min)
	assert.Nil(o.Max)
	assert.InDelta(mean, o.Mean, 0.1)
	assert.InDelta(stdev, o.Stdev, 0.1)

	bsJson, e := json.Marshal(o.Sub(ar))
	assert.NoError(e)
	var bs runningstat.Snapshot
	e = json.Unmarshal(bsJson, &bs)
	assert.NoError(e)
	assert.Equal(br.Count, bs.Count)
	assert.Equal(br.Len, bs.Len)
	assert.Nil(bs.Min)
	assert.Nil(bs.Max)
	assert.InDelta(br.Mean, bs.Mean, 0.1)
	assert.InDelta(br.Stdev, bs.Stdev, 0.1)

	o = o.Scale(10)
	assert.EqualValues(6, o.Count)
	assert.EqualValues(6, o.Len)
	assert.Nil(o.Min)
	assert.Nil(o.Max)
	assert.InDelta(mean*10, o.Mean, 1.0)
	assert.InDelta(stdev*10, o.Stdev, 1.0)
}

func TestIntStat(t *testing.T) {
	assert, _ := makeAR(t)

	var a, b runningstat.IntStat
	a.Init(1)
	b.Init(1)

	o := a.Read()
	assert.EqualValues(0, o.Count)
	assert.EqualValues(0, o.Len)
	assert.Nil(o.Min)
	assert.Nil(o.Max)

	for _, x := range input[:3] {
		a.Push(uint64(x * 10))
	}
	for _, x := range input[3:] {
		b.Push(uint64(x * 10))
	}

	ar, br := a.Read(), b.Read()

	o = ar.Add(br)
	assert.EqualValues(6, o.Count)
	assert.EqualValues(6, o.Len)
	if assert.NotNil(o.Min) {
		assert.EqualValues(min*10, *o.Min)
	}
	if assert.NotNil(o.Max) {
		assert.EqualValues(max*10, *o.Max)
	}
	assert.InDelta(mean*10, o.Mean, 1.0)
	assert.InDelta(stdev*10, o.Stdev, 1.0)
}
