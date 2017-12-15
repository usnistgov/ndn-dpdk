package ndn

import (
	"strings"
	"testing"

	"ndn-traffic-dpdk/dpdk"
)

func TestName(t *testing.T) {
	assert, require := makeAR(t)

	tests := []struct {
		input     string
		ok        bool
		nComps    int
		hasDigest bool
		str       string
	}{
		{"", false, 0, false, ""},
		{"0700", true, 0, false, "/"},
		{"0714 080141 080142 080100 0801FF 800141 0800 08012E", true, 7, false,
			"/A/B/%00/%FF/128=A/.../...."},
		{"0722 0120(DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A)", true, 1, true,
			"/sha256digest=dc6d6840c6fafb773d583cdbf465661c7b4b968e04acd4d9015b1c4e53e59d6a"},
		{"0763 " + strings.Repeat("080141 ", 32) + "080142", true, 33, false,
			strings.Repeat("/A", 32) + "/B"},
		{"0200", false, 0, false, ""},           // bad TLV-TYPE
		{"0704 0102 DDDD", false, 0, false, ""}, // wrong digest length
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		require.True(pkt.IsValid(), tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		n, e := d.ReadName()
		if tt.ok {
			if assert.NoError(e, tt.input) {
				assert.Equal(tt.nComps, n.Len(), tt.input)
				assert.Equal(tt.hasDigest, n.HasDigest(), tt.input)
				assert.Equal(tt.str, n.String(), tt.input)
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}

func TestNameCompare(t *testing.T) {
	assert, require := makeAR(t)

	nameStrs := []string{
		"0700",
		"0702 0200",
		"0702 0800",
		"0704 0800 0800",
		"0703 080141",
		"0705 080141 0800",
		"0707 080141 0800 0800",
		"0706 080141 080141",
		"0703 080142",
		"0704 08024100",
		"0704 08024101",
		"0702 0900",
	}
	pkts := make([]dpdk.Packet, len(nameStrs))
	names := make([]Name, len(nameStrs))
	for i, nameStr := range nameStrs {
		pkts[i] = packetFromHex(nameStr)
		require.True(pkts[i].IsValid(), nameStr)
		d := NewTlvDecoder(pkts[i])
		var e error
		names[i], e = d.ReadName()
		require.NoError(e, nameStr)
	}
	defer func() {
		for _, pkt := range pkts {
			pkt.Close()
		}
	}()

	relTable := [][]int{
		[]int{+0, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1},
		[]int{+1, +0, -2, -2, -2, -2, -2, -2, -2, -2, -2, -2},
		[]int{+1, +2, +0, -1, -2, -2, -2, -2, -2, -2, -2, -2},
		[]int{+1, +2, +1, +0, -2, -2, -2, -2, -2, -2, -2, -2},
		[]int{+1, +2, +2, +2, +0, -1, -1, -1, -2, -2, -2, -2},
		[]int{+1, +2, +2, +2, +1, +0, -1, -2, -2, -2, -2, -2},
		[]int{+1, +2, +2, +2, +1, +1, +0, -2, -2, -2, -2, -2},
		[]int{+1, +2, +2, +2, +1, +2, +2, +0, -2, -2, -2, -2},
		[]int{+1, +2, +2, +2, +2, +2, +2, +2, +0, -2, -2, -2},
		[]int{+1, +2, +2, +2, +2, +2, +2, +2, +2, +0, -2, -2},
		[]int{+1, +2, +2, +2, +2, +2, +2, +2, +2, +2, +0, -2},
		[]int{+1, +2, +2, +2, +2, +2, +2, +2, +2, +2, +2, +0},
	}
	assert.Equal(len(names), len(relTable))
	for i, relRow := range relTable {
		assert.Equal(len(names), len(relRow), i)
		for j, rel := range relRow {
			cmp := names[i].Compare(names[j])
			assert.Equal(NameCompareResult(rel), cmp, "%d=%s %d=%s", i, names[i], j, names[j])
		}
	}
}
