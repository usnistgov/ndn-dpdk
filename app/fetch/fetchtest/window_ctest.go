package fetchtest

/*
#include "../../../csrc/fetch/window.h"
*/
import "C"
import (
	"testing"
)

func ctestWindowSmall(t *testing.T) {
	assert, require := makeAR(t)

	win := &C.FetchWindow{}
	C.FetchWindow_Init(win, 8, C.SOCKET_ID_ANY)
	defer C.FetchWindow_Free(win)

	var seg1 *C.FetchSeg
	for i := 0; i <= 7; i++ {
		seg := C.FetchWindow_Append(win)
		if i == 1 {
			seg1 = seg
		}
		require.NotNil(seg, i)
		assert.EqualValues(i, seg.segNum)
	}
	assert.Nil(C.FetchWindow_Append(win))

	C.FetchWindow_Delete(win, 2)
	C.FetchWindow_Delete(win, 4)
	assert.Equal(seg1, C.FetchWindow_Get(win, 1))
	C.FetchWindow_Delete(win, 1)
	assert.Nil(C.FetchWindow_Get(win, 1))
	C.FetchWindow_Delete(win, 1)
	assert.Nil(C.FetchWindow_Get(win, 1))
	C.FetchWindow_Delete(win, 0)

	for i := 8; i <= 10; i++ {
		seg := C.FetchWindow_Append(win)
		require.NotNil(seg, "%d", i)
		assert.EqualValues(i, seg.segNum)
	}
	assert.Nil(C.FetchWindow_Append(win))
}

func ctestWindowLarge(t *testing.T) {
	assert, require := makeAR(t)

	win := &C.FetchWindow{}
	C.FetchWindow_Init(win, 0x4000, C.SOCKET_ID_ANY)
	defer C.FetchWindow_Free(win)

	appendRange := func(first, last int) {
		for i := first; i <= last; i++ {
			seg := C.FetchWindow_Append(win)
			require.NotNil(seg, i)
			assert.EqualValues(i, seg.segNum)
		}
	}
	deleteRange := func(first, last int) {
		for i := first; i <= last; i++ {
			C.FetchWindow_Delete(win, C.uint64_t(i))
		}
	}

	appendRange(0x0000, 0x3FFF)
	assert.Nil(C.FetchWindow_Append(win))

	deleteRange(0x0000, 0x0001)
	appendRange(0x4000, 0x4001)
	assert.Nil(C.FetchWindow_Append(win))

	deleteRange(0x0002, 0x200D)
	deleteRange(0x200F, 0x4000)
	deleteRange(0x200E, 0x200E)
	appendRange(0x4002, 0x8000)
	assert.Nil(C.FetchWindow_Append(win))
}
