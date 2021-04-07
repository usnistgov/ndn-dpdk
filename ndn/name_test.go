package ndn_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
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

func TestNameSlice(t *testing.T) {
	assert, _ := makeAR(t)

	name := ndn.ParseName("/A/B/C/D/E")
	nameEqual(assert, "/A", ndn.Name{name.Get(0)})
	nameEqual(assert, "/A", ndn.Name{name.Get(-5)})
	nameEqual(assert, "/C", ndn.Name{name.Get(2)})
	nameEqual(assert, "/C", ndn.Name{name.Get(-3)})
	nameEqual(assert, "/E", ndn.Name{name.Get(4)})
	nameEqual(assert, "/E", ndn.Name{name.Get(-1)})
	assert.False(name.Get(-6).Valid())
	assert.False(name.Get(5).Valid())

	nameEqual(assert, "/A", name.Slice(0, 1))
	nameEqual(assert, "/A", name.Slice(0, -4))
	nameEqual(assert, "/A", name.Slice(-5, 1))
	nameEqual(assert, "/A", name.Slice(-5, -4))
	nameEqual(assert, "/B", name.Slice(1, 2))
	nameEqual(assert, "/B", name.Slice(1, -3))
	nameEqual(assert, "/B", name.Slice(-4, 2))
	nameEqual(assert, "/B", name.Slice(-4, -3))
	nameEqual(assert, "/C/D", name.Slice(2, 4))
	nameEqual(assert, "/C/D", name.Slice(2, -1))
	nameEqual(assert, "/C/D", name.Slice(-3, 4))
	nameEqual(assert, "/C/D", name.Slice(-3, -1))
	nameEqual(assert, "/D/E", name.Slice(3, 5))
	nameEqual(assert, "/D/E", name.Slice(3))
	nameEqual(assert, "/D/E", name.Slice(-2, 5))
	nameEqual(assert, "/D/E", name.Slice(-2))
	assert.Len(name.Slice(2, 2), 0)
	assert.Len(name.Slice(2, 0), 0)
	assert.Len(name.Slice(-1, -2), 0)
	assert.Len(name.Slice(-6, 1), 0)
	assert.Len(name.Slice(1, 6), 0)

	nameEqual(assert, "/A/B/C", name.GetPrefix(3))
	nameEqual(assert, "/A/B/C", name.GetPrefix(-2))
	nameEqual(assert, "/A/B/C/D/E", name.GetPrefix(5))
	nameEqual(assert, "/", name.GetPrefix(0))
	assert.Len(name.GetPrefix(6), 0)

	assert.Panics(func() { name.Slice(0, 1, 2) })
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
		{input: "ndn:/", output: "0700", canonical: "/"},
		{input: "/", output: "0700"},
		{input: "/G", output: "0703 080147", canonical: "/8=G"},
		{input: "/8=H/I", output: "0706 080148 080149", canonical: "/8=H/8=I"},
		{input: "/.../..../.....", output: "0709 0800 08012E 08022E2E", canonical: "/8=.../8=..../8=....."},
		{input: "/8=%00GH%ab%cD%EF", output: "0708 0806004748ABCDEF", canonical: "/8=%00GH%AB%CD%EF"},
		{input: "/2=A", output: "0703 020141"},
		{input: "/255=A", output: "0705 FD00FF0141"},
		{input: "/65535=A", output: "0705 FDFFFF0141"},
	}
	for _, tt := range tests {
		name := ndn.ParseName(tt.input)
		wire, e := tlv.EncodeFrom(name)
		assert.NoError(e, tt.input)
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
		assert.True(obj.Name.Equal(jsonDecoded.Name))
		assert.Equal(50, jsonDecoded.I)
	}

	var jsonDecoded2 marshalTestStruct
	assert.Error(json.Unmarshal([]byte(`{"Name":4,"I":50}`), &jsonDecoded2))
	if assert.NoError(json.Unmarshal([]byte(`{"Name":null,"I":50}`), &jsonDecoded2)) {
		assert.Nil(jsonDecoded2.Name)
	}
}
