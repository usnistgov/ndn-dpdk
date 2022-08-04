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
	return tlv.TLVBytes(uint32(m), make([]byte, m))
}

func TestEncode(t *testing.T) {
	assert, _ := makeAR(t)

	wire, e := tlv.EncodeFrom(
		tlv.Bytes(nil),
		tlv.Bytes([]byte{0xF1}),
		tlv.FieldFunc(func(b []byte) ([]byte, error) { return append(b, 0xF2), nil }),
		tlv.TLVBytes(1, []byte{0xF3}),
		testEncodeMarshaler(2),
		tlv.TLV(3, testEncodeMarshaler(3).Field()),
		tlv.TLVFrom(4, testEncodeMarshaler(4)),
	)
	assert.NoError(e)
	assert.Equal([]byte{
		0xF1,
		0xF2,
		0x01, 0x01, 0xF3,
		0x02, 0x02, 0x00, 0x00,
		0x03, 0x05, 0x03, 0x03, 0x00, 0x00, 0x00,
		0x04, 0x06, 0x04, 0x04, 0x00, 0x00, 0x00, 0x00,
	}, wire)

	wire, e = tlv.EncodeValueOnly(tlv.TLVBytes(5, []byte{0xF4}))
	assert.NoError(e)
	assert.Equal([]byte{0xF4}, wire)

	_, e = tlv.Encode(testEncodeMarshaler(-1).Field())
	assert.Error(e)
	_, e = tlv.Encode(tlv.TLVNNI(10, -1))
	assert.Error(e)
	_, e = tlv.EncodeValueOnly(tlv.TLVNNI(11, -1))
	assert.Error(e)
}
