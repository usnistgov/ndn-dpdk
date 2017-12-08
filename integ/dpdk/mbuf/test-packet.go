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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
	"unsafe"
)

func testPacket() {
	assert := assert.New(t)
	require := require.New(t)

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
