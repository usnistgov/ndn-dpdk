package ethdev_test

import (
	"encoding/json"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
)

func TestEtherAddr(t *testing.T) {
	assert, require := makeAR(t)

	a, e := ethdev.ParseEtherAddr("XXXX")
	assert.Error(e)

	a, e = ethdev.ParseEtherAddr("02:00:00:00:AB:01:00:00")
	assert.Error(e)

	a, e = ethdev.ParseEtherAddr("00-00-00-00-00-00")
	require.NoError(e)
	assert.True(a.IsZero())
	assert.False(a.IsUnicast())
	assert.False(a.IsGroup())
	assert.Equal(a.String(), "00:00:00:00:00:00")

	a, e = ethdev.ParseEtherAddr("02:00:00:00:AB:01")
	require.NoError(e)
	assert.False(a.IsZero())
	assert.True(a.IsUnicast())
	assert.False(a.IsGroup())
	assert.Equal(a.String(), "02:00:00:00:ab:01")
	assert.True(a.Equal(a))

	b, e := ethdev.ParseEtherAddr("03:00:00:00:AB:01")
	require.NoError(e)
	assert.False(a.IsZero())
	assert.False(b.IsUnicast())
	assert.True(b.IsGroup())
	assert.Equal(b.String(), "03:00:00:00:ab:01")
	assert.False(a.Equal(b))
}

func TestEtherAddrJson(t *testing.T) {
	assert, require := makeAR(t)

	a, _ := ethdev.ParseEtherAddr("03:00:00:00:AB:01")
	jsonData, e := json.Marshal(a)
	require.NoError(e)
	assert.Equal(string(jsonData), "\"03:00:00:00:ab:01\"")

	e = json.Unmarshal([]byte("5"), &a)
	assert.Error(e)

	e = json.Unmarshal([]byte("\"02:00:00:00:ab:01\""), &a)
	require.NoError(e)
	assert.Equal(a.String(), "02:00:00:00:ab:01")
}
