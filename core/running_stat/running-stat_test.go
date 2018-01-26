package running_stat_test

import (
	"testing"

	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestRunningStat(t *testing.T) {
	assert, _ := dpdktestenv.MakeAR(t)

	s := running_stat.New()

	assert.Equal(0, s.Len())
	assert.InDelta(0.0, s.Min(), 0.1)
	assert.InDelta(0.0, s.Max(), 0.1)
	assert.InDelta(0.0, s.Mean(), 0.1)
	assert.InDelta(0.0, s.Stdev(), 0.1)

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
	for _, x := range input {
		s.Push(x)
	}

	assert.Equal(6, s.Len())
	assert.EqualValues(6, s.Len64())
	assert.InDelta(727.7, s.Min(), 0.1)
	assert.InDelta(1956.1, s.Max(), 0.1)
	assert.InDelta(1285.5, s.Mean(), 0.1)
	assert.InDelta(420.96, s.Stdev(), 0.1)
}
