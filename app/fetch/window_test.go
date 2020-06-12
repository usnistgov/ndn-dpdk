package fetch_test

import (
	"testing"

	"ndn-dpdk/app/fetch"
	"ndn-dpdk/dpdk/eal"
)

func TestWindow(t *testing.T) {
	assert, require := makeAR(t)

	var win fetch.Window
	win.Init(8, eal.NumaSocket{})
	defer win.Close()

	var seg1 *fetch.SegState
	for i := 0; i <= 7; i++ {
		seg := win.Append()
		if i == 1 {
			seg1 = seg
		}
		require.NotNil(seg, "%d", i)
		assert.Equal(i, int(seg.SegNum))
	}
	assert.Nil(win.Append())

	win.Delete(2)
	win.Delete(4)
	assert.Equal(seg1, win.Get(1))
	win.Delete(1)
	assert.Nil(win.Get(1))
	win.Delete(0)

	for i := 8; i <= 10; i++ {
		seg := win.Append()
		require.NotNil(seg, "%d", i)
		assert.Equal(i, int(seg.SegNum))
	}
	assert.Nil(win.Append())
}
