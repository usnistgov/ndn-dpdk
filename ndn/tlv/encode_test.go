package tlv_test

import (
	"errors"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

type testEncodeMarshaler int

func (m testEncodeMarshaler) Field() tlv.Field {
	if m < 0 {
		return tlv.FieldError(errors.New("testEncodeMarshaler error"))
	}
	return tlv.TLVBytes(uint32(m), make([]byte, int(m)))
}

func TestEncode(t *testing.T) {
	assert, _ := makeAR(t)

	wire, e := tlv.EncodeFrom(tlv.Bytes([]byte{0xF1}), testEncodeMarshaler(2), testEncodeMarshaler(3), testEncodeMarshaler(4))
	assert.NoError(e)
	assert.Equal([]byte{
		0xF1,
		0x02, 0x02, 0x00, 0x00,
		0x03, 0x03, 0x00, 0x00, 0x00,
		0x04, 0x04, 0x00, 0x00, 0x00, 0x00,
	}, wire)

	_, e = tlv.Encode(testEncodeMarshaler(-1).Field())
	assert.Error(e)
}
