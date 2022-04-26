package pktmbuf_test

import (
	"bytes"
	"testing"
)

func TestPacketRead(t *testing.T) {
	assert, require := makeAR(t)
	vec := directMp.MustAlloc(2)
	defer vec.Close()

	part0 := bytes.Repeat([]byte{0xA0}, 100)
	part1 := bytes.Repeat([]byte{0xA1}, 200)
	part2 := bytes.Repeat([]byte{0xA2}, 300)

	pkt := vec[0]
	require.NotNil(pkt)
	assert.Equal(0, pkt.Len())
	assert.Equal([][]byte{{}}, pkt.SegmentBytes())

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
	assert.Equal([][]byte{part1, {}}, pkt.SegmentBytes())

	pkt.Append(part2)
	assert.Equal(500, pkt.Len())
	assert.Equal([][]byte{part1, part2}, pkt.SegmentBytes())
	pkt.Prepend(part0)
	assert.Equal(600, pkt.Len())
	assert.Equal([][]byte{bytes.Join([][]byte{part0, part1}, nil), part2}, pkt.SegmentBytes())
}
