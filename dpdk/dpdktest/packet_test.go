package dpdktest

import (
	"bytes"
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

	// subtest name: (segment count)-(delete from which segment)(where in that segment)

	t.Run("1-0head", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3")
		defer pkt.Close()
		pkt.DeleteRange(0, 2)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A2A3"), pkt.ReadAll())
		assert.Equal(1, mp.CountInUse())
	})
	t.Run("1-0mid", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3")
		defer pkt.Close()
		pkt.DeleteRange(1, 2)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A3"), pkt.ReadAll())
		assert.Equal(1, mp.CountInUse())
	})
	t.Run("1-0tail", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3")
		defer pkt.Close()
		pkt.DeleteRange(2, 2)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1"), pkt.ReadAll())
		assert.Equal(1, mp.CountInUse())
	})
	t.Run("1-0all", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3")
		defer pkt.Close()
		pkt.DeleteRange(0, 4)
		assert.Equal(0, pkt.Len())
		assert.Equal(1, mp.CountInUse())
	})
	t.Run("2-0tail", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3")
		defer pkt.Close()
		pkt.DeleteRange(2, 2)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1B0B1B2B3"), pkt.ReadAll())
		assert.Equal(2, mp.CountInUse())
	})
	t.Run("2-0all", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3")
		defer pkt.Close()
		pkt.DeleteRange(0, 4)
		assert.Equal(dpdktestenv.PacketBytesFromHex("B0B1B2B3"), pkt.ReadAll())
		assert.Equal(2, mp.CountInUse())
	})
	t.Run("2-1head", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3")
		defer pkt.Close()
		pkt.DeleteRange(4, 2)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1A2A3B2B3"), pkt.ReadAll())
		assert.Equal(2, mp.CountInUse())
	})
	t.Run("2-1all", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3")
		defer pkt.Close()
		pkt.DeleteRange(4, 4)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1A2A3"), pkt.ReadAll())
		assert.Equal(1, mp.CountInUse())
	})
	t.Run("2-0tail-1head", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3")
		defer pkt.Close()
		pkt.DeleteRange(2, 4)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1B2B3"), pkt.ReadAll())
		assert.Equal(2, mp.CountInUse())
	})
	t.Run("2-0tail-1all", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3")
		defer pkt.Close()
		pkt.DeleteRange(2, 6)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1"), pkt.ReadAll())
		assert.Equal(1, mp.CountInUse())
	})
	t.Run("3-1all", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3", "C0C1C2C3")
		defer pkt.Close()
		pkt.DeleteRange(4, 4)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1A2A3C0C1C2C3"), pkt.ReadAll())
		assert.Equal(2, mp.CountInUse())
	})
	t.Run("3-0tail-1all", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3", "C0C1C2C3")
		defer pkt.Close()
		pkt.DeleteRange(2, 6)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1C0C1C2C3"), pkt.ReadAll())
		assert.Equal(2, mp.CountInUse())
	})
	t.Run("3-1all-2head", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3", "C0C1C2C3")
		defer pkt.Close()
		pkt.DeleteRange(4, 6)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1A2A3C2C3"), pkt.ReadAll())
		assert.Equal(2, mp.CountInUse())
	})
	t.Run("3-0tail-1all-2head", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3", "C0C1C2C3")
		defer pkt.Close()
		pkt.DeleteRange(2, 8)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1C2C3"), pkt.ReadAll())
		assert.Equal(2, mp.CountInUse())
	})
	t.Run("3-0tail-1all-2all", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := dpdktestenv.PacketFromHex("A0A1A2A3", "B0B1B2B3", "C0C1C2C3")
		defer pkt.Close()
		pkt.DeleteRange(2, 10)
		assert.Equal(dpdktestenv.PacketBytesFromHex("A0A1"), pkt.ReadAll())
		assert.Equal(1, mp.CountInUse())
	})
}
