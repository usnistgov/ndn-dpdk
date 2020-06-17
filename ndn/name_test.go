package ndn_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestNameDecode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input  string
		bad    bool
		nComps int
	}{
		{input: "", nComps: 0},
		{input: "08F0 DDDD", bad: true},
		{input: "FE0001000000", bad: true},
		{input: "080141 080142 080100 0801FF 800141 0800 08012E", nComps: 7},
		{input: strings.Repeat("080141 ", 32) + "080142", nComps: 33},
		{input: "0120(DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A)", nComps: 1},
	}
	for _, tt := range tests {
		var name ndn.Name
		e := name.UnmarshalBinary(bytesFromHex(tt.input))
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			assert.Equal(tt.nComps, len(name), tt.input)
		}
	}
}

func TestNameCompare(t *testing.T) {
	assert, _ := makeAR(t)

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
	names := make([]ndn.Name, len(nameStrs))
	for i, nameStr := range nameStrs {
		names[i].UnmarshalBinary(bytesFromHex(nameStr))
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
			switch rel {
			case -2:
				assert.Less(cmp, 0, "%d=%s %d=%s", i, names[i], j, names[j])
				assert.False(names[i].IsPrefixOf(names[j]), "%d=%s %d=%s", i, names[i], j, names[j])
				assert.False(names[j].IsPrefixOf(names[i]), "%d=%s %d=%s", i, names[i], j, names[j])
			case -1:
				assert.Less(cmp, 0, "%d=%s %d=%s", i, names[i], j, names[j])
				assert.True(names[i].IsPrefixOf(names[j]), "%d=%s %d=%s", i, names[i], j, names[j])
				assert.False(names[j].IsPrefixOf(names[i]), "%d=%s %d=%s", i, names[i], j, names[j])
			case 0:
				assert.Zero(cmp, "%d=%s %d=%s", i, names[i], j, names[j])
				assert.True(names[i].IsPrefixOf(names[j]), "%d=%s %d=%s", i, names[i], j, names[j])
				assert.True(names[j].IsPrefixOf(names[i]), "%d=%s %d=%s", i, names[i], j, names[j])
			case +1:
				assert.Greater(cmp, 0, "%d=%s %d=%s", i, names[i], j, names[j])
				assert.False(names[i].IsPrefixOf(names[j]), "%d=%s %d=%s", i, names[i], j, names[j])
				assert.True(names[j].IsPrefixOf(names[i]), "%d=%s %d=%s", i, names[i], j, names[j])
			case +2:
				assert.Greater(cmp, 0, "%d=%s %d=%s", i, names[i], j, names[j])
				assert.False(names[i].IsPrefixOf(names[j]), "%d=%s %d=%s", i, names[i], j, names[j])
				assert.False(names[j].IsPrefixOf(names[i]), "%d=%s %d=%s", i, names[i], j, names[j])
			}
		}
	}
}

func TestNameParse(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		output    string
		canonical string
	}{
		{input: "ndn:/", output: "", canonical: "/"},
		{input: "/", output: ""},
		{input: "/G", output: "080147", canonical: "/8=G"},
		{input: "/8=H/I", output: "080148 080149", canonical: "/8=H/8=I"},
		{input: "/.../..../.....", output: "0800 08012E 08022E2E", canonical: "/8=.../8=..../8=....."},
		{input: "/8=%00GH%ab%cD%EF", output: "0806004748ABCDEF", canonical: "/8=%00GH%AB%CD%EF"},
		{input: "/2=A", output: "020141"},
		{input: "/255=A", output: "FD00FF0141"},
		{input: "/65535=A", output: "FDFFFF0141"},
	}
	for _, tt := range tests {
		name := ndn.ParseName(tt.input)
		wire, _ := name.MarshalBinary()
		bytesEqual(assert, bytesFromHex(tt.output), wire, tt.input)
		if tt.canonical == "" {
			tt.canonical = tt.input
		}
		assert.Equal(tt.canonical, name.String(), tt.input)
	}
}

type marshalTestStruct struct {
	Name ndn.Name
	I    int
}

func TestNameMarshal(t *testing.T) {
	assert, _ := makeAR(t)

	var obj marshalTestStruct
	obj.Name = ndn.ParseName("/A/B")
	obj.I = 50

	jsonEncoding, e := json.Marshal(obj)
	if assert.NoError(e) {
		assert.Equal([]byte(`{"Name":"/8=A/8=B","I":50}`), jsonEncoding)
	}

	var jsonDecoded marshalTestStruct
	if e := json.Unmarshal(jsonEncoding, &jsonDecoded); assert.NoError(e) {
		assert.Zero(obj.Name.Compare(jsonDecoded.Name))
		assert.Equal(50, jsonDecoded.I)
	}

	var jsonDecoded2 marshalTestStruct
	assert.Error(json.Unmarshal([]byte(`{"Name":4,"I":50}`), &jsonDecoded2))
	if assert.NoError(json.Unmarshal([]byte(`{"Name":null,"I":50}`), &jsonDecoded2)) {
		assert.Nil(jsonDecoded2.Name)
	}
}
