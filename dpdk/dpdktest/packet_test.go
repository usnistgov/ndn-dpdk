package dpdktest

import (
	"bytes"
	"strings"
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestPacketSegmentsRead(t *testing.T) {
	assert, require := makeAR(t)
	dpdktestenv.MakeDirectMp(63, 0, 1000)

	pkt := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	defer pkt.Close()
	assert.Equal(0, pkt.Len())
	assert.Equal(1, pkt.CountSegments())
	assert.Equal(pkt.GetFirstSegment(), pkt.GetLastSegment())
	seg0p := pkt.GetSegment(0)
	require.NotNil(seg0p)
	assert.Equal(pkt.GetFirstSegment(), *seg0p)

	pkt.GetFirstSegment().Append(bytes.Repeat([]byte{0xA1}, 200))
	assert.Equal(200, pkt.Len())

	seg1, e := pkt.AppendSegment(dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT))
	require.NoError(e)
	assert.Equal(pkt.GetLastSegment(), seg1)
	seg1p := pkt.GetSegment(1)
	require.NotNil(seg1p)
	assert.Equal(seg1, *seg1p)

	seg1.Append(bytes.Repeat([]byte{0xA2}, 300))
	assert.Equal(500, pkt.Len())

	readTests := []struct {
		offset   int
		expected []byte
	}{
		{0, []byte{0xA1, 0xA1, 0xA1, 0xA1}},
		{196, []byte{0xA1, 0xA1, 0xA1, 0xA1}},
		{197, []byte{0xA1, 0xA1, 0xA1, 0xA2}},
		{199, []byte{0xA1, 0xA2, 0xA2, 0xA2}},
		{200, []byte{0xA2, 0xA2, 0xA2, 0xA2}},
		{496, []byte{0xA2, 0xA2, 0xA2, 0xA2}},
		{498, []byte{0xA2, 0xA2}},
		{500, []byte{}},
	}
	for _, tt := range readTests {
		readBuf := make([]byte, 4)
		nRead := pkt.ReadTo(tt.offset, readBuf)
		assert.Equal(len(tt.expected), nRead, tt.offset)
		assert.Equal(tt.expected, readBuf[:nRead], tt.offset)
	}
}

func TestPacketClone(t *testing.T) {
	assert, require := makeAR(t)
	mp := dpdktestenv.MakeDirectMp(63, 0, 1000)
	mpi := dpdktestenv.MakeIndirectMp(63)

	var pkt0mbufs [2]dpdk.Mbuf
	dpdktestenv.AllocBulk(dpdktestenv.MPID_DIRECT, pkt0mbufs[:])
	pkt0 := pkt0mbufs[0].AsPacket()
	pkt0.AppendSegment(pkt0mbufs[1])
	pkt0.GetFirstSegment().Append(bytes.Repeat([]byte{0xA1}, 100))
	pkt0.GetLastSegment().Append(bytes.Repeat([]byte{0xA2}, 200))
	assert.Equal(2, mp.CountInUse())

	pkt1, e := mpi.ClonePkt(pkt0)
	require.NoError(e)
	require.True(pkt1.IsValid())
	assert.Equal(2, mp.CountInUse())
	assert.Equal(2, mpi.CountInUse())
	assert.Equal(2, pkt1.CountSegments())
	assert.Equal(300, pkt1.Len())
	assert.Equal(pkt0.GetFirstSegment().GetData(), pkt1.GetFirstSegment().GetData())
	assert.Equal(pkt0.GetLastSegment().GetData(), pkt1.GetLastSegment().GetData())

	pkt0.Close()
	assert.Equal(2, mp.CountInUse())
	assert.Equal(2, mpi.CountInUse())

	pkt1.Close()
	assert.Equal(0, mp.CountInUse())
	assert.Equal(0, mpi.CountInUse())
}

func TestPacketDeleteRange(t *testing.T) {
	mp := dpdktestenv.MakeDirectMp(63, 0, 4)

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
			pkt := dpdktestenv.PacketFromHex(strings.Split(tt.pkt, "/")...)
			defer pkt.Close()
			expected := pkt.ReadAll()
			expected = append(expected[:tt.offset], expected[tt.offset+tt.count:]...)

			begin := dpdk.NewPacketIterator(pkt)
			pi := begin
			pi.Advance(tt.offset)
			pkt.DeleteRange(&pi, tt.count)

			assert.Equal(tt.offset, begin.ComputeDistance(pi))
			assert.Equal(expected, pkt.ReadAll())
			assert.Equal(tt.nSegs, mp.CountInUse())
		})
	}
}

func TestPacketLinearizeRange(t *testing.T) {
	mp := dpdktestenv.MakeDirectMp(63, 0, 4)

	getSegmentLengths := func(pkt dpdk.Packet) (lens []int) {
		for seg, ok := pkt.GetFirstSegment(), true; ok; seg, ok = seg.GetNext() {
			lens = append(lens, seg.Len())
		}
		return lens
	}

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
			pkt := dpdktestenv.PacketFromHex(strings.Split(tt.pkt, "/")...)
			defer pkt.Close()
			payload := pkt.ReadAll()

			begin := dpdk.NewPacketIterator(pkt)
			first := begin
			first.Advance(tt.first)
			last := begin
			last.Advance(tt.last)

			linear, e := pkt.LinearizeRange(&first, &last, mp)
			require.NoError(e)
			require.Equal(tt.segLen, getSegmentLengths(pkt))
			assert.Equal(uintptr(pkt.GetSegment(tt.inSeg).GetData())+tt.atOffset, uintptr(linear))
			assert.Equal(payload, pkt.ReadAll())
			assert.Equal(len(tt.segLen), mp.CountInUse())
			assert.Equal(tt.first, begin.ComputeDistance(first))
			assert.Equal(tt.last, begin.ComputeDistance(last))
		})
	}
}
