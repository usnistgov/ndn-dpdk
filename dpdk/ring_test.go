package dpdk

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"unsafe"
)

func TestRingObjTable(t *testing.T) {
	ot := NewRingObjTable(4)
	defer ot.Close()

	ot.Set(0, unsafe.Pointer(uintptr(6698)))
	ot.Set(1, unsafe.Pointer(uintptr(3110)))

	res := testRingObjTable(ot)
	assert.Equal(t, 0, res, "testCArgs C function error")

	assert.Equal(t, unsafe.Pointer(uintptr(4891)), ot.Get(3))
}
