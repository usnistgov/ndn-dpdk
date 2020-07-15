package cptrtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func TestByteSlice(t *testing.T) {
	assert, _ := makeAR(t)

	var byteArray [5]byte
	byteArray[2] = 0xA0
	b := cptr.AsByteSlice(&byteArray)
	assert.Len(b, 5)
	assert.EqualValues(0xA0, b[2])
	b[3] = 0xA1
	assert.EqualValues(0xA1, byteArray[3])

	var int8Array [6]int8
	int8Array[2] = 0x20
	b = cptr.AsByteSlice(&int8Array)
	assert.Len(b, 6)
	assert.EqualValues(0x20, b[2])
	b[3] = 0x21
	assert.EqualValues(0x21, int8Array[3])

	int8Slice := make([]int8, 7)
	int8Slice[2] = 0x30
	b = cptr.AsByteSlice(&int8Slice)
	assert.Len(b, 7)
	assert.EqualValues(0x30, b[2])
	b[3] = 0x31
	assert.EqualValues(0x31, int8Slice[3])

	var emptyArray [0]byte
	b = cptr.AsByteSlice(&emptyArray)
	assert.Len(b, 0)

	assert.Panics(func() { cptr.AsByteSlice(byteArray) })
	var int32Array [1]int32
	assert.Panics(func() { cptr.AsByteSlice(&int32Array) })
}
