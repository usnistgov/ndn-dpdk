package pktmbuf_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
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
	assert.Equal(200, pkt.GetHeadroom())
	tail0 := pkt.GetTailroom()
	pkt.Append(part1)
	assert.Equal(200, pkt.Len())
	assert.Equal(200, tail0-pkt.GetTailroom())

	seg1 := vec[1]
	e := pkt.Chain(seg1)
	require.NoError(e)
	vec[1] = nil // avoid double-free during vec.Close()
	assert.True(pkt.IsSegmented())

	pkt.Append(part2)
	assert.Equal(500, pkt.Len())
	pkt.Prepend(part0)
	assert.Equal(600, pkt.Len())

	assert.Equal(bytes.Join([][]byte{part0, part1, part2}, nil), pkt.ReadAll())
}

func TestPacketDeleteRange(t *testing.T) {
	tests := []struct {
		id     string // (segment count)-(delete from which segment)(where in that segment)
		pkt    string // packet segments, separated by '/'
		offset int
		count  int
		nSegs  int
	}{
		{"1-0head", "A0A1A2A3", 0, 2, 1},
		{"1-0mid", "A0A1A2A3", 1, 2, 1},
		{"1-0tail", "A0A1A2A3", 1, 2, 1},
		{"1-0all", "A0A1A2A3", 0, 4, 1},
		{"2-0tail", "A0A1A2A3/B0B1B2B3", 2, 2, 2},
		{"2-0all", "A0A1A2A3/B0B1B2B3", 0, 4, 2},
		{"2-1head", "A0A1A2A3/B0B1B2B3", 4, 2, 2},
		{"2-1all", "A0A1A2A3/B0B1B2B3", 4, 4, 1},
		{"2-0tail-1head", "A0A1A2A3/B0B1B2B3", 2, 4, 2},
		{"2-0tail-1all", "A0A1A2A3/B0B1B2B3", 2, 6, 1},
		{"3-1all", "A0A1A2A3/B0B1B2B3/C0C1C2C3", 4, 4, 2},
		{"3-0tail-1all", "A0A1A2A3/B0B1B2B3/C0C1C2C3", 2, 6, 2},
		{"3-1all-2head", "A0A1A2A3/B0B1B2B3/C0C1C2C3", 4, 6, 2},
		{"3-0tail-1all", "A0A1A2A3/B0B1B2B3/C0C1C2C3", 2, 6, 2},
		{"3-0tail-1all-2head", "A0A1A2A3/B0B1B2B3/C0C1C2C3", 2, 8, 2},
		{"3-0tail-1all-2all", "A0A1A2A3/B0B1B2B3/C0C1C2C3", 2, 10, 1},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert, _ := makeAR(t)
			pkt := mbuftestenv.MakePacket(strings.Split(tt.pkt, "/"))
			defer pkt.Close()
			expected := pkt.ReadAll()
			expected = append(expected[:tt.offset], expected[tt.offset+tt.count:]...)

			pkt.DeleteRange(tt.offset, tt.count)

			assert.Equal(expected, pkt.ReadAll())
			assert.Len(mbuftestenv.ListSegmentLengths(pkt), tt.nSegs)
		})
	}
}

func TestPacketLinearizeRange(t *testing.T) {
	mp, _ := pktmbuf.NewPool("TEST_TINY4", pktmbuf.PoolConfig{Capacity: 63, Dataroom: 4}, eal.NumaSocket{})
	tests := []struct {
		id       string
		pkt      string // packet segments, separated by '/'
		first    int
		last     int
		segLen   []int
		inSeg    int
		atOffset uintptr
	}{
		{"InSeg", "A0A1A2A3/B0B1B2B3", 1, 3, []int{4, 4}, 0, 1},
		{"AppendToFirst-KeepLastSeg", "A0A1/B0/C0C1", 1, 4, []int{4, 1}, 0, 1},
		{"AppendToFirst-FreeLastSeg", "A0A1/B0/C0", 1, 4, []int{4}, 0, 1},
		{"NewSeg-KeepLastSeg", "A0A1A2A3/B0/C0C1C2", 3, 6, []int{3, 3, 2}, 1, 0},
		{"NewSeg-FreeLastSeg", "A0A1A2A3/B0/C0", 3, 6, []int{3, 3}, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert, require := makeAR(t)
			pkt := mbuftestenv.MakePacket(mp, mbuftestenv.Headroom(0), strings.Split(tt.pkt, "/"))
			defer pkt.Close()
			payload := pkt.ReadAll()

			e := pkt.LinearizeRange(tt.first, tt.last, mp)
			require.NoError(e)
			assert.Equal(tt.segLen, mbuftestenv.ListSegmentLengths(pkt))
			assert.Equal(payload, pkt.ReadAll())
		})
	}
}
