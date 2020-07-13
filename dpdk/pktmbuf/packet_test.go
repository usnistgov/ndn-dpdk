package pktmbuf_test

import (
	"bytes"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

func TestPacketRead(t *testing.T) {
	assert, require := makeAR(t)
	vec := mbuftestenv.Direct.Pool().MustAlloc(2)
	defer vec.Close()

	part0 := bytes.Repeat([]byte{0xA0}, 100)
	part1 := bytes.Repeat([]byte{0xA1}, 200)
	part2 := bytes.Repeat([]byte{0xA2}, 300)

	pkt := vec[0]
	require.NotNil(pkt)
	assert.Equal(0, pkt.Len())
	assert.False(pkt.IsSegmented())

	pkt.SetHeadroom(200)
	assert.Equal(200, pkt.Headroom())
	tail0 := pkt.Tailroom()
	pkt.Append(part1)
	assert.Equal(200, pkt.Len())
	assert.Equal(200, tail0-pkt.Tailroom())

	seg1 := vec[1]
	e := pkt.Chain(seg1)
	require.NoError(e)
	vec[1] = nil // avoid double-free during vec.Close()
	assert.True(pkt.IsSegmented())

	pkt.Append(part2)
	assert.Equal(500, pkt.Len())
	pkt.Prepend(part0)
	assert.Equal(600, pkt.Len())

	assert.Equal(bytes.Join([][]byte{part0, part1, part2}, nil), pkt.Bytes())
}
