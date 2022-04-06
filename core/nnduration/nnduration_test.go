package nnduration_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var (
	makeAR = testenv.MakeAR
)

func TestMilliseconds(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Equal(2816*time.Millisecond, nnduration.Milliseconds(0).DurationOr(2816))

	ms := nnduration.Milliseconds(5274)
	assert.Equal(5274*time.Millisecond, ms.DurationOr(2816))
	assert.Equal(`5274`, testenv.ToJSON(ms))

	assert.Equal(ms, testenv.FromJSON[nnduration.Milliseconds](`5274`))

	assert.Equal(ms, testenv.FromJSON[nnduration.Milliseconds](`"5274"`))

	decoded := testenv.FromJSON[nnduration.Milliseconds](`"6s"`)
	assert.Equal(nnduration.Milliseconds(6000), decoded)
	assert.Equal(6*time.Second, decoded.Duration())
}

func TestNanoseconds(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Equal(1652*time.Nanosecond, nnduration.Nanoseconds(0).DurationOr(1652))

	ns := nnduration.Nanoseconds(7011)
	assert.Equal(7011*time.Nanosecond, ns.DurationOr(1652))
	assert.Equal(`7011`, testenv.ToJSON(ns))

	assert.Equal(ns, testenv.FromJSON[nnduration.Nanoseconds](`7011`))

	decoded := testenv.FromJSON[nnduration.Nanoseconds](`"3us"`)
	assert.Equal(nnduration.Nanoseconds(3000), decoded)
	assert.Equal(3*time.Microsecond, decoded.Duration())
}
