package main

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()
	assert := assert.New(t)
	require := require.New(t)

	_, e := dpdk.NewEal([]string{"testprog", "-n1"})
	require.NoError(e)

	mp, e := dpdk.NewPktmbufPool("MP", 63, 0, 0, 1000, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	require.NotNil(mp)
	defer mp.Close()
	assert.EqualValues(63, mp.CountAvailable())
	assert.EqualValues(0, mp.CountInUse())

	var mbufs [63]dpdk.Mbuf
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

	m0, e := mp.Alloc()
	require.NoError(e)
	assert.EqualValues(0, m0.GetDataLength())
	assert.True(m0.GetHeadroom() > 0)
	assert.True(m0.GetTailroom() > 0)
	e = m0.SetHeadroom(200)
	require.NoError(e)
	assert.EqualValues(200, m0.GetHeadroom())
	assert.EqualValues(800, m0.GetTailroom())

	m0p1, e := m0.Prepend(100)
	require.NoError(e)
	C.memset(m0p1, 0xA1, 100)
	m0p2, e := m0.Append(200)
	require.NoError(e)
	C.memset(m0p2, 0xA2, 200)
	assert.EqualValues(300, m0.GetDataLength())
	assert.EqualValues(100, m0.GetHeadroom())
	assert.EqualValues(600, m0.GetTailroom())

	allocBuf := C.malloc(4)
	defer C.free(allocBuf)
	readBuf := m0.Read(98, 4, allocBuf)
	assert.Equal([]byte{0xA1, 0xA1, 0xA2, 0xA2}, C.GoBytes(readBuf, 4))

	m0p3, e := m0.Adj(50)
	require.NoError(e)
	assert.EqualValues(50, uintptr(m0p3)-uintptr(m0p1))
	e = m0.Trim(50)
	require.NoError(e)
	assert.EqualValues(200, m0.GetDataLength())
	assert.EqualValues(150, m0.GetHeadroom())
	assert.EqualValues(650, m0.GetTailroom())

	_, e = m0.Prepend(151)
	assert.Error(e)
	_, e = m0.Append(651)
	assert.Error(e)
	_, e = m0.Adj(201)
	assert.Error(e)
	e = m0.Trim(201)
	assert.Error(e)
}
