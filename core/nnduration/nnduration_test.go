package nnduration_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/nnduration"
)

func TestMilliseconds(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Equal(2816*time.Millisecond, nnduration.Milliseconds(0).DurationOr(2816))

	ms := nnduration.Milliseconds(5274)
	assert.Equal(5274*time.Millisecond, ms.DurationOr(2816))

	j, e := json.Marshal(ms)
	assert.NoError(e)
	assert.Equal(([]byte)("5274"), j)

	var decoded nnduration.Milliseconds
	e = json.Unmarshal(j, &decoded)
	assert.NoError(e)
	assert.Equal(ms, decoded)

	e = json.Unmarshal(([]byte)("\"5274\""), &decoded)
	assert.NoError(e)
	assert.Equal(ms, decoded)

	e = json.Unmarshal(([]byte)(`"6s"`), &decoded)
	assert.NoError(e)
	assert.Equal(nnduration.Milliseconds(6000), decoded)
	assert.Equal(6*time.Second, decoded.Duration())
}

func TestNanoseconds(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Equal(1652*time.Nanosecond, nnduration.Nanoseconds(0).DurationOr(1652))

	ns := nnduration.Nanoseconds(7011)
	assert.Equal(7011*time.Nanosecond, ns.DurationOr(1652))

	j, e := json.Marshal(ns)
	assert.NoError(e)
	assert.Equal(([]byte)("7011"), j)

	var decoded nnduration.Nanoseconds
	e = json.Unmarshal(j, &decoded)
	assert.NoError(e)
	assert.Equal(ns, decoded)

	e = json.Unmarshal(([]byte)(`"3us"`), &decoded)
	assert.NoError(e)
	assert.Equal(nnduration.Nanoseconds(3000), decoded)
	assert.Equal(3*time.Microsecond, decoded.Duration())
}
