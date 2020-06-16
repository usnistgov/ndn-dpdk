package ndni_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
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
		{input: "08F0 DDDD", err: ndni.NdnError_Incomplete},
		{input: "FE0001000000", err: ndni.NdnError_BadNameComponentType},
		{input: "080141 080142 080100 0801FF 800141 0800 08012E", nComps: 7},
		{input: strings.Repeat("080141 ", 32) + "080142", nComps: 33},
		{input: "0120(DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A)",
			nComps: 1, hasDigest: true},
		{input: "0102 DDDD", err: ndni.NdnError_BadDigestComponentLength},
	}
	for _, tt := range tests {
		n, e := ndni.NewName(tlvBytesFromHex(tt.input))
		if tt.bad || tt.err != nil {
			assert.Error(e, tt.input)
			if tt.err != nil {
				assert.EqualError(e, tt.err.Error(), tt.input)
			}
		} else if assert.NoError(e, tt.input) {
			assert.Equal(tt.nComps, n.Len(), tt.input)
			assert.Equal(tt.hasDigest, n.HasDigestComp(), tt.input)
		}
	}
}

func TestNamePrefixHash(t *testing.T) {
	assert, require := makeAR(t)

	input := tlvBytesFromHex("080141 080142 080100 0801FF 800141 0800 08012E" +
		strings.Repeat(" 080141", 32))
	n, e := ndni.NewName(input)
	require.NoError(e)
	require.Equal(39, n.Len())
	assert.Equal(len(input), n.Size())

	hashes := make(map[uint64]bool)
	for i := 0; i < 39; i++ {
		prefix := n.GetPrefix(i)
		require.NoError(e)
		assert.Equal(i, prefix.Len())
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
	names := make([]*ndni.Name, len(nameStrs))
	for i, nameStr := range nameStrs {
		var e error
		names[i], e = ndni.NewName(tlvBytesFromHex(nameStr))
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
			assert.Equal(ndni.NameCompareResult(rel), cmp, "%d=%s %d=%s", i, names[i], j, names[j])
			if rel == 0 {
				assert.True(names[i].Equal(names[j]), "%d=%s %d=%s", i, names[i], j, names[j])
			} else {
				assert.False(names[i].Equal(names[j]), "%d=%s %d=%s", i, names[i], j, names[j])
			}
		}
	}
}

func TestNameParse(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		bad       bool
		output    string
		canonical string
	}{
		{input: "ndn:/", output: "", canonical: "/"},
		{input: "/", output: ""},
		{input: "/G", output: "080147", canonical: "/8=G"},
		{input: "/8=H/I", output: "080148 080149", canonical: "/8=H/8=I"},
		{input: "/.../..../.....", output: "0800 08012E 08022E2E", canonical: "/8=.../8=..../8=....."},
		{input: "//A", bad: true},
		{input: "/.", bad: true},
		{input: "/..", bad: true},
		{input: "/8=%00GH%ab%cD%EF", output: "0806004748ABCDEF", canonical: "/8=%00GH%AB%CD%EF"},
		{input: "/2=A", output: "020141"},
		{input: "/255=A", output: "FD00FF0141"},
		{input: "/65535=A", output: "FDFFFF0141"},
		{input: "/0=A", bad: true},
		{input: "/65536=A", bad: true},
		{input: "/hello=A", bad: true},
	}
	for _, tt := range tests {
		n, e := ndni.ParseName(tt.input)
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			expected := ndni.TlvBytes(mbuftestenv.BytesFromHex(tt.output))
			assert.Equal(expected, n.GetValue(), tt.input)
			if tt.canonical == "" {
				tt.canonical = tt.input
			}
			assert.Equal(tt.canonical, n.String(), tt.input)
		}
	}
}

type marshalTestStruct struct {
	Name *ndni.Name
	I    int
}

func TestNameMarshal(t *testing.T) {
	assert, _ := makeAR(t)

	var obj marshalTestStruct
	obj.Name = ndni.MustParseName("/A/B")
	obj.I = 50

	jsonEncoding, e := json.Marshal(obj)
	if assert.NoError(e) {
		assert.Equal([]byte(`{"Name":"/8=A/8=B","I":50}`), jsonEncoding)
	}

	var jsonDecoded marshalTestStruct
	if e := json.Unmarshal(jsonEncoding, &jsonDecoded); assert.NoError(e) {
		assert.True(obj.Name.Equal(jsonDecoded.Name))
		assert.Equal(50, jsonDecoded.I)
	}

	var jsonDecoded2 marshalTestStruct
	assert.Error(json.Unmarshal([]byte(`{"Name":4,"I":50}`), &jsonDecoded2))
	if assert.NoError(json.Unmarshal([]byte(`{"Name":null,"I":50}`), &jsonDecoded2)) {
		assert.Nil(jsonDecoded2.Name)
	}
}
