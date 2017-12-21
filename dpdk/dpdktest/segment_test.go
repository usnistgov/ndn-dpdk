package dpdktest

import (
	"bytes"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestSegment(t *testing.T) {
	assert, require := makeAR(t)
	mp := dpdktestenv.MakeDirectMp(63, 0, 1000)

	m, e := mp.Alloc()
	require.NoError(e)

	pkt := m.AsPacket()
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
	c_memset(dp1, 0xA1, 100)
	dp2, e := s.Append(150)
	require.NoError(e)
	c_memset(dp2, 0xA2, 150)
	e = s.AppendOctets(bytes.Repeat([]byte{0xA3}, 50))
	require.NoError(e)
	assert.EqualValues(300, s.Len())
	assert.EqualValues(100, s.GetHeadroom())
	assert.EqualValues(600, s.GetTailroom())

	assert.Equal(append(append(bytes.Repeat([]byte{0xA1}, 100), bytes.Repeat([]byte{0xA2}, 150)...), bytes.Repeat([]byte{0xA3}, 50)...),
		c_GoBytes(s.GetData(), s.Len()))

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
