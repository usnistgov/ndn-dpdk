package dpdk

import (
	"bytes"
	"testing"
	"unsafe"
)

func TestMbuf(t *testing.T) {
	_, require := makeAR(t)

	mp, e := NewPktmbufPool("MP", 63, 0, 0, 1000, NUMA_SOCKET_ANY)
	require.NoError(e)
	require.NotNil(mp)
	defer mp.Close()

	t.Run("Mempool", func(t *testing.T) {
		assert, _ := makeAR(t)

		assert.EqualValues(63, mp.CountAvailable())
		assert.EqualValues(0, mp.CountInUse())

		var mbufs [63]Mbuf
		e = mp.AllocBulk(mbufs[30:])
		assert.NoError(e)
		assert.EqualValues(30, mp.CountAvailable())
		assert.EqualValues(33, mp.CountInUse())
		for i := 0; i < 30; i++ {
			mbufs[i], e = mp.Alloc()
			assert.NoError(e)
		}
		assert.EqualValues(0, mp.CountAvailable())
		assert.EqualValues(63, mp.CountInUse())
		_, e = mp.Alloc()
		assert.Error(e)
		mbufs[0].Close()
		assert.EqualValues(1, mp.CountAvailable())
		assert.EqualValues(62, mp.CountInUse())

		for i := 1; i < 63; i++ {
			mbufs[i].Close()
		}
		assert.EqualValues(63, mp.CountAvailable())
	})

	t.Run("Segment", func(t *testing.T) {
		assert, require := makeAR(t)

		m, e := mp.Alloc()
		require.NoError(e)

		pkt := Packet{m}
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
		dp2, e := s.Append(200)
		require.NoError(e)
		c_memset(dp2, 0xA2, 200)
		assert.EqualValues(300, s.Len())
		assert.EqualValues(100, s.GetHeadroom())
		assert.EqualValues(600, s.GetTailroom())

		assert.Equal(append(bytes.Repeat([]byte{0xA1}, 100), bytes.Repeat([]byte{0xA2}, 200)...),
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
	})

	t.Run("Packet", func(t *testing.T) {
		assert, require := makeAR(t)

		m, e := mp.Alloc()
		require.NoError(e)
		var cMbufPtr *c_struct_rte_mbuf
		assert.Equal(unsafe.Sizeof(cMbufPtr), unsafe.Sizeof(m))

		pkt := Packet{m}
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

		it0 := NewPacketIterator(pkt)
		readSuccessTests := []struct {
			offset         uint
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
			it.Advance(tt.offset)
			require.Falsef(it.IsEnd(), "it.Advance(%d) is past end", tt.offset)
			assert.EqualValuesf(-int(tt.offset), it.ComputeDistance(&it0),
				"it.Advance(%d).ComputeDistance(it0) is wrong", tt.offset)
			assert.EqualValuesf(tt.offset, it0.ComputeDistance(&it),
				"it0.ComputeDistance(it.Advance(%d)) is wrong", tt.offset)

			readBuf2 := make([]byte, 4)
			nRead := it.Read(readBuf2[:])
			assert.EqualValuesf(4, nRead, "%d it.Read() has wrong length", tt.offset)
			assert.Equalf(tt.expected[:], readBuf2, "%d it.Read() returns wrong bytes", tt.offset)
		}

		_, e = pkt.Read(497, 4, allocBuf)
		assert.Error(e)

		m, e = mp.Alloc()
		require.NoError(e)
		pkt2 := Packet{m}
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
	})

	t.Run("PacketClone", func(t *testing.T) {
		assert, require := makeAR(t)

		pkts := make([]Packet, 2)
		e = mp.AllocPktBulk(pkts[:1])
		require.NoError(e)

		m, e := mp.Alloc()
		require.NoError(e)
		pkts[0].AppendSegment(m, nil)
		pkts[0].GetFirstSegment().Append(100)
		pkts[0].GetLastSegment().Append(200)
		assert.EqualValues(2, mp.CountInUse())

		mpi, e := NewPktmbufPool("MP-INDIRECT", 63, 0, 0, 0, NUMA_SOCKET_ANY)
		require.NoError(e)
		require.NotNil(mpi)
		defer mpi.Close()

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
	})
}
