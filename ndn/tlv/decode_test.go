package tlv_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestDecode(t *testing.T) {
	assert, _ := makeAR(t)

	d := tlv.DecodingBuffer(bytesFromHex("F100 F20120 1F023031 01"))

	elements := d.Elements()
	assert.Len(elements, 3)

	assert.EqualValues(0xF1, elements[0].Type)
	assert.Len(elements[0].Value, 0)
	assert.True(elements[0].IsCriticalType())
	assert.Len(elements[0].Wire, 2)
	assert.Len(elements[0].After, 8)

	assert.EqualValues(0xF2, elements[1].Type)
	assert.Len(elements[1].Value, 1)
	assert.False(elements[1].IsCriticalType())
	assert.Len(elements[1].Wire, 3)
	assert.Len(elements[1].After, 5)

	assert.EqualValues(0x1F, elements[2].Type)
	assert.Len(elements[2].Value, 2)
	assert.True(elements[2].IsCriticalType())
	assert.Len(elements[2].Wire, 4)
	assert.Len(elements[2].After, 1)

	var element1 tlv.Element
	assert.NoError(elements[1].Unmarshal(&element1))
	var nni2 tlv.NNI
	assert.NoError(elements[2].UnmarshalValue(&nni2))
	assert.EqualValues(0x3031, nni2)

	assert.Len(d.Rest(), 1)
	assert.False(d.EOF())
	assert.Error(d.ErrUnlessEOF())
}
