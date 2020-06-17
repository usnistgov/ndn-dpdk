package ndni_test

import (
	"strings"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestNameDecode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		bad       bool
		err       error
		nComps    int
		hasDigest bool
	}{
		{input: "", nComps: 0},
		{input: "08F0 DDDD", err: ndni.NdnErrIncomplete},
		{input: "FE0001000000", err: ndni.NdnErrBadNameComponentType},
		{input: "080141 080142 080100 0801FF 800141 0800 08012E", nComps: 7},
		{input: strings.Repeat("080141 ", 32) + "080142", nComps: 33},
		{input: "0120(DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A)",
			nComps: 1, hasDigest: true},
		{input: "0102 DDDD", err: ndni.NdnErrBadDigestComponentLength},
	}
	for _, tt := range tests {
		n, e := ndni.NewCName(bytesFromHex(tt.input))
		if tt.bad || tt.err != nil {
			assert.Error(e, tt.input)
			if tt.err != nil {
				assert.EqualError(e, tt.err.Error(), tt.input)
			}
		} else if assert.NoError(e, tt.input) {
			assert.EqualValues(tt.nComps, n.P.NComps, tt.input)
			assert.Equal(tt.hasDigest, n.P.HasDigestComp, tt.input)
		}
	}
}

func TestNamePrefixHash(t *testing.T) {
	assert, require := makeAR(t)

	input := bytesFromHex("080141 080142 080100 0801FF 800141 0800 08012E" +
		strings.Repeat(" 080141", 32))
	n, e := ndni.NewCName(input)
	require.NoError(e)
	require.EqualValues(39, n.P.NComps)
	assert.EqualValues(len(input), n.P.NOctets)

	hashes := make(map[uint64]bool)
	for i := 0; i < 39; i++ {
		prefix := ndni.CNameFromName(n.ToName()[:i])
		assert.EqualValues(i, prefix.P.NComps)
		hash := prefix.ComputeHash()
		hashes[hash] = true
		assert.Equal(hash, n.ComputePrefixHash(i))
	}

	assert.InDelta(39, len(hashes), 9.1) // expect at least 30 different hash values
}

func TestNameCompare(t *testing.T) {
	assert, require := makeAR(t)

	nameStrs := []string{
		"",
		"0200",
		"0800",
		"0800 0800",
		"080141",
		"080141 0800",
		"080141 0800 0800",
		"080141 080141",
		"080142",
		"08024100",
		"08024101",
		"0900",
	}
	names := make([]*ndni.CName, len(nameStrs))
	for i, nameStr := range nameStrs {
		var e error
		names[i], e = ndni.NewCName(bytesFromHex(nameStr))
		require.NoError(e, nameStr)
	}

	relTable := [][]int{
		{+0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		{+1, +0, -2, -2, -2, -2, -2, -2, -2, -2, -2, -2},
		{+1, +2, +0, -1, -2, -2, -2, -2, -2, -2, -2, -2},
		{+1, +2, +1, +0, -2, -2, -2, -2, -2, -2, -2, -2},
		{+1, +2, +2, +2, +0, -1, -1, -1, -2, -2, -2, -2},
		{+1, +2, +2, +2, +1, +0, -1, -2, -2, -2, -2, -2},
		{+1, +2, +2, +2, +1, +1, +0, -2, -2, -2, -2, -2},
		{+1, +2, +2, +2, +1, +2, +2, +0, -2, -2, -2, -2},
		{+1, +2, +2, +2, +2, +2, +2, +2, +0, -2, -2, -2},
		{+1, +2, +2, +2, +2, +2, +2, +2, +2, +0, -2, -2},
		{+1, +2, +2, +2, +2, +2, +2, +2, +2, +2, +0, -2},
		{+1, +2, +2, +2, +2, +2, +2, +2, +2, +2, +2, +0},
	}
	assert.Equal(len(names), len(relTable))
	for i, relRow := range relTable {
		assert.Equal(len(names), len(relRow), i)
		for j, rel := range relRow {
			cmp := names[i].Compare(names[j])
			assert.Equal(rel, cmp, "%d=%s %d=%s", i, names[i], j, names[j])
		}
	}
}
