package cptrtest

/*
#include <stdint.h>
*/
import "C"
import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
)

func ctestByteSlice(t *testing.T) {
	assert, _ := makeAR(t)

	var charArray [6]C.char
	b := cptr.AsByteSlice(charArray[:0])
	assert.Len(b, 0)

	charArray[2] = 0x30
	b = cptr.AsByteSlice(charArray[:])
	assert.Len(b, 6)
	assert.EqualValues(0x30, b[2])
	b[3] = 0x31
	assert.EqualValues(0x31, charArray[3])

	uint8Slice := make([]C.uint8_t, 7)
	uint8Slice[2] = 0x40
	b = cptr.AsByteSlice(uint8Slice)
	assert.Len(b, 7)
	assert.EqualValues(0x40, b[2])
	b[3] = 0x41
	assert.EqualValues(0x41, uint8Slice[3])
}
