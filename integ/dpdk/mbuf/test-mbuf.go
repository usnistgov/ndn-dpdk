package main

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <stdlib.h>
#include <string.h>
#include <rte_config.h>
#include <rte_mbuf.h>
*/
import "C"
import (
	"bytes"
	"unsafe"
	assertPkg "github.com/stretchr/testify/assert"
	requirePkg "github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
)

var t *integ.Testing
var mp dpdk.PktmbufPool
var assert *assertPkg.Assertions
var require *requirePkg.Assertions

func main() {
	t = new(integ.Testing)
	defer t.Close()
	assert = assertPkg.New(t)
	require = requirePkg.New(t)

	_, e := dpdk.NewEal([]string{"testprog", "-n1"})
	require.NoError(e)

	mp, e = dpdk.NewPktmbufPool("MP", 63, 0, 0, 1000, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	require.NotNil(mp)
	defer mp.Close()

	testMempool()
	testSegment()
	testPacket()
}

func testMempool() {
	assert.EqualValues(63, mp.CountAvailable())
	assert.EqualValues(0, mp.CountInUse())

	var mbufs [63]dpdk.Mbuf
	e := mp.AllocBulk(mbufs[30:])
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
}

func testSegment() {
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

func testPacket() {
	m, e := mp.Alloc()
	require.NoError(e)
	var cMbufPtr *C.struct_rte_mbuf
	assert.Equal(unsafe.Sizeof(cMbufPtr), unsafe.Sizeof(m))

	pkt := dpdk.Packet{m}
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
	C.memset(dp0, 0xA1, 200)
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
	C.memset(dp1, 0xA2, 300)
	assert.EqualValues(500, pkt.Len())

	allocBuf := C.malloc(4)
	defer C.free(allocBuf)

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
			assert.Truef(allocBuf != readBuf, "Read(%d) is using allocBuf", tt.offset)
		}
		assert.Equalf(tt.expected[:], C.GoBytes(readBuf, 4),
			"Read(%d) returns wrong bytes", tt.offset)
	}

	_, e = pkt.Read(497, 4, allocBuf)
	assert.Error(e)

	m, e = mp.Alloc()
	require.NoError(e)
	pkt2 := dpdk.Packet{m}
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
