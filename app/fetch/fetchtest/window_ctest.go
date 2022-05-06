package fetchtest

/*
#include "../../../csrc/fetch/window.h"
*/
import "C"
import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/fetch"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func ctestWindow(t *testing.T) {
	assert, require := makeAR(t)

	var win fetch.Window
	win.Init(8, eal.NumaSocket{})
	defer win.Close()
	winC := (*C.FetchWindow)(unsafe.Pointer(&win))

	var seg1 *C.FetchSeg
	for i := 0; i <= 7; i++ {
		seg := C.FetchWindow_Append(winC)
		if i == 1 {
			seg1 = seg
		}
		require.NotNil(seg, "%d", i)
		assert.EqualValues(i, seg.segNum)
	}
	assert.Nil(C.FetchWindow_Append(winC))

	C.FetchWindow_Delete(winC, 2)
	C.FetchWindow_Delete(winC, 4)
	assert.Equal(seg1, C.FetchWindow_Get(winC, 1))
	C.FetchWindow_Delete(winC, 1)
	assert.Nil(C.FetchWindow_Get(winC, 1))
	C.FetchWindow_Delete(winC, 0)

	for i := 8; i <= 10; i++ {
		seg := C.FetchWindow_Append(winC)
		require.NotNil(seg, "%d", i)
		assert.EqualValues(i, seg.segNum)
	}
	assert.Nil(C.FetchWindow_Append(winC))
}
