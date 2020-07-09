package nnduration_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
)

func TestMilliseconds(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Equal(2816*time.Millisecond, nnduration.Milliseconds(0).DurationOr(2816))

	ms := nnduration.Milliseconds(5274)
	assert.Equal(5274*time.Millisecond, ms.DurationOr(2816))
	assert.Equal(`5274`, toJSON(ms))

	var decoded nnduration.Milliseconds
	fromJSON(`5274`, &decoded)
	assert.Equal(ms, decoded)

	fromJSON(`"5274"`, &decoded)
	assert.Equal(ms, decoded)

	fromJSON(`"6s"`, &decoded)
	assert.Equal(nnduration.Milliseconds(6000), decoded)
	assert.Equal(6*time.Second, decoded.Duration())
}

func TestNanoseconds(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Equal(1652*time.Nanosecond, nnduration.Nanoseconds(0).DurationOr(1652))

	ns := nnduration.Nanoseconds(7011)
	assert.Equal(7011*time.Nanosecond, ns.DurationOr(1652))
	assert.Equal(`7011`, toJSON(ns))

	var decoded nnduration.Nanoseconds
	fromJSON(`7011`, &decoded)
	assert.Equal(ns, decoded)

	fromJSON(`"3us"`, &decoded)
	assert.Equal(nnduration.Nanoseconds(3000), decoded)
	assert.Equal(3*time.Microsecond, decoded.Duration())
}
