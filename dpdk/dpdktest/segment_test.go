package dpdktest

import (
	"bytes"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestSegment(t *testing.T) {
	assert, _ := makeAR(t)
	dpdktestenv.MakeDirectMp(63, 0, 1000)

	pkt := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	defer pkt.Close()
	s := pkt.GetFirstSegment()

	assert.Equal(0, s.Len())
	assert.NotNil(s.GetData())
	assert.True(s.GetHeadroom() > 0)
	assert.True(s.GetTailroom() > 0)
	assert.NoError(s.SetHeadroom(200))
	assert.Equal(200, s.GetHeadroom())
	assert.Equal(800, s.GetTailroom())

	c1 := bytes.Repeat([]byte{0xA1}, 100)
	c2 := bytes.Repeat([]byte{0xA2}, 150)
	assert.NoError(s.Prepend(c1))
	assert.NoError(s.Append(c2))
	assert.Equal(250, s.Len())
	assert.Equal(100, s.GetHeadroom())
	assert.Equal(650, s.GetTailroom())
	assert.Equal(append(append([]byte{}, c1...), c2...), c_GoBytes(s.GetData(), s.Len()))

	assert.NoError(s.Adj(50))
	assert.NoError(s.Trim(100))
	assert.Equal(100, s.Len())
	assert.Equal(150, s.GetHeadroom())
	assert.Equal(750, s.GetTailroom())
	assert.Equal(append(append([]byte{}, c1[50:]...), c2[:50]...), c_GoBytes(s.GetData(), s.Len()))

	assert.Error(s.Prepend(bytes.Repeat([]byte{0xA0}, 151)))
	assert.Error(s.Append(bytes.Repeat([]byte{0xA0}, 751)))
	assert.Error(s.Adj(101))
	assert.Error(s.Trim(101))
}
