package nnduration_test

import (
	"encoding/json"
	"testing"
	"time"

	"ndn-dpdk/core/nnduration"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestMilliseconds(t *testing.T) {
	assert, _ := dpdktestenv.MakeAR(t)

	ms := nnduration.Milliseconds(5274)

	j, e := json.Marshal(ms)
	assert.NoError(e)
	assert.Equal(([]byte)("5274"), j)

	var decoded nnduration.Milliseconds
	e = json.Unmarshal(j, &decoded)
	assert.NoError(e)
	assert.Equal(ms, decoded)

	e = json.Unmarshal(([]byte)(`"6s"`), &decoded)
	assert.NoError(e)
	assert.Equal(nnduration.Milliseconds(6000), decoded)
	assert.Equal(6*time.Second, decoded.Duration())
}

func TestNanoseconds(t *testing.T) {
	assert, _ := dpdktestenv.MakeAR(t)

	ms := nnduration.Nanoseconds(7011)

	j, e := json.Marshal(ms)
	assert.NoError(e)
	assert.Equal(([]byte)("7011"), j)

	var decoded nnduration.Nanoseconds
	e = json.Unmarshal(j, &decoded)
	assert.NoError(e)
	assert.Equal(ms, decoded)

	e = json.Unmarshal(([]byte)(`"3us"`), &decoded)
	assert.NoError(e)
	assert.Equal(nnduration.Nanoseconds(3000), decoded)
	assert.Equal(3*time.Microsecond, decoded.Duration())
}
