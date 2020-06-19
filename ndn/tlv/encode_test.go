package tlv_test

import (
	"errors"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

type testEncodeMarshaler int

func (m testEncodeMarshaler) MarshalTlv() (typ uint32, value []byte, e error) {
	if m < 0 {
		return 0, nil, errors.New("testEncodeMarshaler error")
	}
	return uint32(m), make([]byte, int(m)), nil
}

func TestEncode(t *testing.T) {
	assert, _ := makeAR(t)

	wire, e := tlv.Encode([]byte{0xF1}, testEncodeMarshaler(2), []testEncodeMarshaler{3, 4})
	assert.NoError(e)
	assert.Equal([]byte{
		0xF1,
		0x02, 0x02, 0x00, 0x00,
		0x03, 0x03, 0x00, 0x00, 0x00,
		0x04, 0x04, 0x00, 0x00, 0x00, 0x00,
	}, wire)

	wire, e = tlv.Encode(testEncodeMarshaler(-1))
	assert.Error(e)
}
