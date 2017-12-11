package ndn

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestVarNum(t *testing.T) {
	assert := assert.New(t)

	buf := make([]byte, VARNUM_BUFLEN)

	encodeDecodeTests := []struct {
		n   uint64
		len uint
	}{
		{0, 1},
		{38, 1},
		{252, 1},
		{253, 3},
		{256, 3},
		{10423, 3},
		{65535, 3},
		{65536, 5},
		{240530981, 5},
		{4294967295, 5},
		{4294967296, 9},
		{18826124832703, 9},
		{18446744073709551615, 9},
	}
	for _, tt := range encodeDecodeTests {
		assert.EqualValuesf(tt.len, EncodeVarNum(tt.n, buf), "%d", tt.n)
		n, len := DecodeVarNum(buf)
		assert.EqualValuesf(tt.n, n, "%d", tt.n)
		assert.EqualValuesf(tt.len, len, "%d", tt.n)
	}
}
