package cptrtest

/*
#include <stdlib.h>
#include <spdk/env.h>
*/
import "C"
import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

// As of Go 1.17.8 + gcc 7 + SPDK 22.01, calling an SPDK function significantly reduces linker execution time.
var _ = C.spdk_get_ticks()

func ctestCptrArray(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Panics(func() { cptr.ParseCptrArray(1) })
	assert.Panics(func() { cptr.ParseCptrArray("x") })
	assert.Panics(func() { cptr.ParseCptrArray([]string{"x", "y"}) })

	_, count := cptr.ParseCptrArray([](*C.int){})
	assert.Equal(0, count)

	int0, int1 := C.malloc(C.sizeof_int), C.malloc(C.sizeof_int)
	defer C.free(int0)
	defer C.free(int1)
	*(*C.int)(int0) = 0xAAA1
	*(*C.int)(int1) = 0xAAA2

	ptr, count := cptr.ParseCptrArray([]*C.int{(*C.int)(int0), (*C.int)(int1)})
	assert.Equal(2, count)
	assert.EqualValues(0xAAA1, **(**C.int)(ptr))
	assert.EqualValues(0xAAA2, **(**C.int)(unsafe.Add(ptr, unsafe.Sizeof(int0))))
}
