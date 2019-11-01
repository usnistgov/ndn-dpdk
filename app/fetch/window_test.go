package fetch_test

import (
	"testing"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/dpdk"
)

func TestWindow(t *testing.T) {
	assert, require := makeAR(t)

	var win fetch.FetchWindow
	win.Init(8, dpdk.NUMA_SOCKET_ANY)
	defer win.Close()

	var seg1 *fetch.FetchSeg
	for i := 0; i <= 7; i++ {
		segNum, seg := win.Append()
		if i == 1 {
			seg1 = seg
		}
		require.NotNil(seg, "%d", i)
		assert.Equal(i, int(segNum))
	}
	_, noSeg := win.Append()
	assert.Nil(noSeg)

	win.Delete(2)
	win.Delete(4)
	assert.Equal(seg1, win.Get(1))
	win.Delete(1)
	assert.Nil(win.Get(1))
	win.Delete(0)

	for i := 8; i <= 10; i++ {
		segNum, seg := win.Append()
		require.NotNil(seg, "%d", i)
		assert.Equal(i, int(segNum))
	}
	_, noSeg = win.Append()
	assert.Nil(noSeg)
}
