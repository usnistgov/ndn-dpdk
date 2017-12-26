package dpdktest

import (
	"testing"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestPacket(t *testing.T) {
	assert, require := makeAR(t)
	mp := dpdktestenv.MakeDirectMp(63, 0, 1000)

	m, e := mp.Alloc()
	require.NoError(e)
	var cMbufPtr *c_struct_rte_mbuf
	assert.Equal(unsafe.Sizeof(cMbufPtr), unsafe.Sizeof(m))

	pkt := m.AsPacket()
	defer pkt.Close()
	assert.Equal(unsafe.Sizeof(cMbufPtr), unsafe.Sizeof(pkt))

	assert.EqualValues(0, pkt.Len())
	assert.EqualValues(1, pkt.CountSegments())
	assert.Equal(pkt.GetFirstSegment(), pkt.GetLastSegment())
	seg0, e := pkt.GetSegment(0)
	assert.NoError(e)
	assert.Equal(pkt.GetFirstSegment(), seg0)
	_, e = pkt.GetSegment(1)
	assert.Error(e)

	dp0, e := seg0.Append(200)
	c_memset(dp0, 0xA1, 200)
	assert.EqualValues(200, pkt.Len())

	m, e = mp.Alloc()
	require.NoError(e)
	seg1, e := pkt.AppendSegment(m, nil)
	require.NoError(e)
	assert.EqualValues(200, pkt.Len())
	assert.EqualValues(2, pkt.CountSegments())
	assert.NotEqual(pkt.GetFirstSegment(), pkt.GetLastSegment())
	seg1, e = pkt.GetSegment(1)
	assert.NoError(e)
	assert.Equal(pkt.GetLastSegment(), seg1)

	dp1, e := seg1.Append(300)
	c_memset(dp1, 0xA2, 300)
	assert.EqualValues(500, pkt.Len())

	allocBuf := c_malloc(4)
	defer c_free(allocBuf)

	it0 := dpdk.NewPacketIterator(pkt)
	readBuf2 := make([]byte, 4)
	readSuccessTests := []struct {
		offset         int
		shouldUseAlloc bool
		expected       [4]byte
	}{
		{0, false, [4]byte{0xA1, 0xA1, 0xA1, 0xA1}},
		{196, false, [4]byte{0xA1, 0xA1, 0xA1, 0xA1}},
		{197, true, [4]byte{0xA1, 0xA1, 0xA1, 0xA2}},
		{199, true, [4]byte{0xA1, 0xA2, 0xA2, 0xA2}},
		{200, false, [4]byte{0xA2, 0xA2, 0xA2, 0xA2}},
		{496, false, [4]byte{0xA2, 0xA2, 0xA2, 0xA2}},
	}
	for _, tt := range readSuccessTests {
		readBuf, e := pkt.Read(tt.offset, 4, allocBuf)
		assert.NoErrorf(e, "Read(%d) has error", tt.offset)
		require.NotNilf(readBuf, "Read(%d) returns nil", tt.offset)
		if tt.shouldUseAlloc {
			assert.Equalf(allocBuf, readBuf, "Read(%d) is not using allocBuf", tt.offset)
		} else {
			assert.NotEqualf(allocBuf, readBuf, "Read(%d) is using allocBuf", tt.offset)
		}
		assert.Equalf(tt.expected[:], c_GoBytes(readBuf, 4),
			"Read(%d) returns wrong bytes", tt.offset)

		it := it0
		nAdvanced := it.Advance(tt.offset)
		require.Falsef(it.IsEnd(), "it.Advance(%d) is past end", tt.offset)
		assert.EqualValuesf(tt.offset, nAdvanced, "it.Advance(%d) is incomplete", tt.offset)
		assert.EqualValuesf(-int(tt.offset), it.ComputeDistance(&it0),
			"it.Advance(%d).ComputeDistance(it0) is wrong", tt.offset)
		assert.EqualValuesf(tt.offset, it0.ComputeDistance(&it),
			"it0.ComputeDistance(it.Advance(%d)) is wrong", tt.offset)

		assert.EqualValuesf(tt.expected[0], it.PeekOctet(), "%d it.PeekOctet() is wrong", tt.offset)
		nRead := it.Read(readBuf2[:])
		assert.EqualValuesf(4, nRead, "%d it.Read() has wrong length", tt.offset)
		assert.Equalf(tt.expected[:], readBuf2, "%d it.Read() returns wrong bytes", tt.offset)
	}

	it2 := dpdk.NewPacketIteratorBounded(pkt, 100, 6)
	assert.EqualValues(4, it2.Read(readBuf2[:]))
	assert.EqualValues(2, it2.Read(readBuf2[:]))
	assert.EqualValues(0, it2.Read(readBuf2[:]))
	assert.Equal(-1, it2.PeekOctet())

	it2 = dpdk.NewPacketIteratorBounded(pkt, 495, 6)
	assert.EqualValues(3, it2.Advance(3))
	assert.EqualValues(2, it2.Advance(3))
	assert.EqualValues(0, it2.Advance(3))

	_, e = pkt.Read(497, 4, allocBuf)
	assert.Error(e)

	m, e = mp.Alloc()
	require.NoError(e)
	pkt2 := m.AsPacket()
	seg0 = pkt2.GetFirstSegment()
	seg0.Append(50)
	m, e = mp.Alloc()
	require.NoError(e)
	seg1, e = pkt2.AppendSegment(m, &seg0)
	assert.NoError(e)
	seg1.Append(20)
	assert.EqualValues(70, pkt2.Len())
	assert.EqualValues(2, pkt2.CountSegments())

	pkt.AppendPacket(pkt2, nil)
	assert.EqualValues(570, pkt.Len())
	assert.EqualValues(4, pkt.CountSegments())
}

func TestPacketClone(t *testing.T) {
	assert, require := makeAR(t)
	mp := dpdktestenv.MakeDirectMp(63, 0, 1000)
	mpi := dpdktestenv.MakeIndirectMp(63)

	pkts := make([]dpdk.Packet, 2)
	e := mp.AllocPktBulk(pkts[:1])
	require.NoError(e)

	m, e := mp.Alloc()
	require.NoError(e)
	pkts[0].AppendSegment(m, nil)
	pkts[0].GetFirstSegment().Append(100)
	pkts[0].GetLastSegment().Append(200)
	assert.EqualValues(2, mp.CountInUse())

	pkts[1], e = mpi.ClonePkt(pkts[0])
	require.NoError(e)
	require.NotNil(pkts[1])
	assert.EqualValues(2, mp.CountInUse())
	assert.EqualValues(2, mpi.CountInUse())
	assert.EqualValues(2, pkts[1].CountSegments())
	assert.EqualValues(300, pkts[1].Len())
	assert.Equal(pkts[0].GetFirstSegment().GetData(), pkts[1].GetFirstSegment().GetData())
	assert.Equal(pkts[0].GetLastSegment().GetData(), pkts[1].GetLastSegment().GetData())

	pkts[0].Close()
	assert.EqualValues(2, mp.CountInUse())
	assert.EqualValues(2, mpi.CountInUse())

	pkts[1].Close()
	assert.EqualValues(0, mp.CountInUse())
	assert.EqualValues(0, mpi.CountInUse())
}
