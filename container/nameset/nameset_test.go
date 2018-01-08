package nameset_test

import (
	"testing"

	"ndn-dpdk/container/nameset"
	"ndn-dpdk/ndn"
)

func TestNameSet(t *testing.T) {
	assert, _ := makeAR(t)

	set := nameset.New()
	defer set.Close()
	assert.Equal(0, set.Len())
	assert.Equal(-1, set.FindExact(compsFromUri("/")))
	assert.Equal(-1, set.FindPrefix(compsFromUri("/")))

	set.Insert(compsFromUri("/A/B"))
	assert.Equal(1, set.Len())
	assert.True(set.FindPrefix(compsFromUri("/A")) < 0)
	assert.True(set.FindExact(compsFromUri("/A/B")) >= 0)
	assert.True(set.FindPrefix(compsFromUri("/A/B")) >= 0)
	assert.True(set.FindPrefix(compsFromUri("/A/B/C")) >= 0)

	set.Insert(compsFromUri("/C/D"))
	assert.Equal(2, set.Len())
	assert.True(set.FindExact(compsFromUri("/A/B")) >= 0)
	assert.True(set.FindExact(compsFromUri("/C/D")) >= 0)

	set.Erase(set.FindExact(compsFromUri("/C/D")))
	assert.Equal(1, set.Len())
	assert.True(set.FindExact(compsFromUri("/A/B")) >= 0)
	assert.True(set.FindExact(compsFromUri("/C/D")) < 0)
}

func compsFromUri(uri string) ndn.TlvBytes {
	buf, e := ndn.EncodeNameComponentsFromUri(uri)
	if e != nil {
		panic(e)
	}
	return buf
}
