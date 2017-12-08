package main

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <string.h>
*/
import "C"
import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
)

func testSegment() {
	assert := assert.New(t)
	require := require.New(t)

	m, e := mp.Alloc()
	require.NoError(e)

	pkt := dpdk.Packet{m}
	defer pkt.Close()
	s := pkt.GetFirstSegment()

	assert.EqualValues(0, s.Len())
	assert.NotNil(s.GetData())
	assert.True(s.GetHeadroom() > 0)
	assert.True(s.GetTailroom() > 0)
	e = s.SetHeadroom(200)
	require.NoError(e)
	assert.EqualValues(200, s.GetHeadroom())
	assert.EqualValues(800, s.GetTailroom())

	dp1, e := s.Prepend(100)
	require.NoError(e)
	C.memset(dp1, 0xA1, 100)
	dp2, e := s.Append(200)
	require.NoError(e)
	C.memset(dp2, 0xA2, 200)
	assert.EqualValues(300, s.Len())
	assert.EqualValues(100, s.GetHeadroom())
	assert.EqualValues(600, s.GetTailroom())

	assert.Equal(append(bytes.Repeat([]byte{0xA1}, 100), bytes.Repeat([]byte{0xA2}, 200)...),
		C.GoBytes(s.GetData(), C.int(s.Len())))

	dp3, e := s.Adj(50)
	require.NoError(e)
	assert.EqualValues(50, uintptr(dp3)-uintptr(dp1))
	e = s.Trim(50)
	require.NoError(e)
	assert.EqualValues(200, s.Len())
	assert.EqualValues(150, s.GetHeadroom())
	assert.EqualValues(650, s.GetTailroom())

	_, e = s.Prepend(151)
	assert.Error(e)
	_, e = s.Append(651)
	assert.Error(e)
	_, e = s.Adj(201)
	assert.Error(e)
	e = s.Trim(201)
	assert.Error(e)
}
