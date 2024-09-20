package ndnitest

/*
#include "../../csrc/ndni/packet.h"
*/
import "C"
import (
	"bytes"
	"strings"
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func ctestLpParse(t *testing.T) {
	assert, _ := makeAR(t)

	for _, tt := range ndntestvector.LpDecodeTests {
		p := makePacket(tt.Input)
		defer p.Close()

		var lph C.LpHeader
		ok := bool(C.LpHeader_Parse(&lph, p.mbuf))

		if tt.Bad || tt.PayloadL == 0 {
			assert.False(ok, tt.Input)
		} else if assert.True(ok, tt.Input) {
			C.rte_mbuf_sanity_check(p.mbuf, 1)
			assert.EqualValues(tt.SeqNum, C.LpL2_GetSeqNum(&lph.l2), tt.Input)
			assert.EqualValues(tt.FragIndex, lph.l2.fragIndex, tt.Input)
			assert.EqualValues(max(1, tt.FragCount), lph.l2.fragCount, tt.Input)
			if len(tt.PitToken) == 0 {
				assert.Zero(lph.l3.pitToken.length, tt.Input)
			} else {
				assert.Equal(tt.PitToken, cptr.AsByteSlice(lph.l3.pitToken.value[:])[:lph.l3.pitToken.length], tt.Input)
			}
			assert.EqualValues(tt.NackReason, lph.l3.nackReason, tt.Input)
			assert.EqualValues(tt.CongMark, lph.l3.congMark, tt.Input)
			assert.EqualValues(tt.PayloadL, p.Len(), tt.Input)
		}
	}
}

func ctestLpParseTruncate(t *testing.T) {
	assert, require := makeAR(t)
	for _, tt := range []struct {
		Input   string
		SegLens []int
	}{
		// 1-octet payload
		{"6407 pittoken=6202A0A1 payload=5001C0", []int{1}},
		{"6407 pittoken=6202A0A1 payload=5001C0 trailer=F0F1F2F3", []int{1}},
		{"6407 pittoken=6202 / A0A1 payload=5001C0", []int{1}},
		{"6407 pittoken=6202 / A0A1 payload=5001C0 trailer=F0F1F2F3", []int{1}},
		{"6407 pittoken=6202 / A0A1 payload=5001C0 / trailer=F0F1F2F3", []int{1}},
		{"6407 pittoken=6202A0A1 payload=5001 / C0", []int{1}},
		{"6407 pittoken=6202A0A1 payload=5001 / C0 trailer=F0F1F2F3", []int{1}},
		{"6407 pittoken=6202 / A0A1 payload=5001 / C0", []int{1}},
		{"6407 pittoken=6202 / A0A1 payload=5001 / C0 trailer=F0F1F2F3", []int{1}},
		// 4-octet unsegmented payload
		{"640A pittoken=6202A0A1 payload=5004C0C1C2C3", []int{4}},
		{"640A pittoken=6202A0A1 payload=5004C0C1C2C3 trailer=F0F1F2F3", []int{4}},
		{"640A pittoken=6202A0A1 payload=5004C0C1C2C3 / trailer=F0F1F2F3", []int{4}},
		{"640A pittoken=6202A0A1 payload=5004C0C1C2C3 trailer=F0F1 / F2F3", []int{4}},
		{"640A pittoken=6202 / A0A1 payload=5004C0C1C2C3", []int{1, 3}},
		{"640A pittoken=6202 / A0A1 payload=5004C0C1C2C3 trailer=F0F1F2F3", []int{1, 3}},
		{"640A pittoken=6202 / A0A1 payload=5004C0C1C2C3 / trailer=F0F1F2F3", []int{1, 3}},
		{"640A pittoken=6202A0A1 payload=5004 / C0C1C2C3", []int{1, 3}},
		{"640A pittoken=6202A0A1 payload=5004 / C0C1C2C3 trailer=F0F1F2F3", []int{1, 3}},
		{"640A pittoken=6202 / A0A1 payload=5004 / C0C1C2C3", []int{1, 3}},
		{"640A pittoken=6202 / A0A1 payload=5004 / C0C1C2C3 trailer=F0F1F2F3", []int{1, 3}},
		// 4-octet segmented payload
		{"640A pittoken=6202A0A1 payload=5004C0C1C2 / C3", []int{3, 1}},
		{"640A pittoken=6202A0A1 payload=5004C0C1C2 / C3 trailer=F0F1F2F3", []int{3, 1}},
		{"640A pittoken=6202 / A0A1 payload=5004C0C1C2 / C3", []int{1, 2, 1}},
		{"640A pittoken=6202 / A0A1 payload=5004C0C1C2 / C3 trailer=F0F1F2F3", []int{1, 2, 1}},
		{"640A pittoken=6202 / A0A1 payload=5004C0C1C2 / C3 / trailer=F0F1F2F3", []int{1, 2, 1}},
		{"640A pittoken=6202A0A1 payload=5004C0 / C1C2C3", []int{1, 3}},
		{"640A pittoken=6202A0A1 payload=5004C0 / C1C2C3 trailer=F0F1F2F3", []int{1, 3}},
		{"640A pittoken=6202 / A0A1 payload=5004C0 / C1C2C3", []int{1, 3}},
		{"640A pittoken=6202 / A0A1 payload=5004C0 / C1C2C3 trailer=F0F1F2F3", []int{1, 3}},
		{"640A pittoken=6202 / A0A1 payload=5004C0 / C1C2C3 / trailer=F0F1F2F3", []int{1, 3}},
	} {
		p := makePacket(strings.Split(tt.Input, "/"))
		defer p.Close()

		var lph C.LpHeader
		ok := bool(C.LpHeader_Parse(&lph, p.mbuf))
		require.True(ok, tt.Input)
		C.rte_mbuf_sanity_check(p.mbuf, 1)
		segs := p.Packet.SegmentBytes()
		if assert.Len(segs, len(tt.SegLens), tt.Input) {
			for i, seg := range segs {
				assert.Len(seg, tt.SegLens[i], "%s %i", tt.Input, i)
			}
		}
	}
}

func ctestPacketClone(t *testing.T) {
	assert, require := makeAR(t)
	mp := ndnitestenv.MakeMempools()

	data := ndn.MakeData("/D", bytes.Repeat([]byte{0xC0}, 1200))
	wire, _ := tlv.EncodeFrom(data)
	p := makePacket(wire)
	defer p.Close()

	single := toPacket(unsafe.Pointer(p.N.Clone(mp, ndni.PacketTxAlign{
		Linearize:           true,
		FragmentPayloadSize: 7000,
	})))
	require.NotNil(single)
	defer single.Close()
	assert.EqualValues(len(wire), *single.mbufA.PktLen())
	assert.EqualValues(1, *single.mbufA.NbSegs())
	assert.EqualValues(len(wire), *single.mbufA.DataLen())

	linearFrag := toPacket(unsafe.Pointer(p.N.Clone(mp, ndni.PacketTxAlign{
		Linearize:           true,
		FragmentPayloadSize: 500,
	})))
	require.NotNil(linearFrag)
	defer linearFrag.Close()
	assert.EqualValues(len(wire), *linearFrag.mbufA.PktLen())
	assert.EqualValues(3, *linearFrag.mbufA.NbSegs())
	assert.EqualValues(500, *linearFrag.mbufA.DataLen())

	chained := toPacket(unsafe.Pointer(p.N.Clone(mp, ndni.PacketTxAlign{
		Linearize: false,
	})))
	require.NotNil(chained)
	defer chained.Close()
	assert.EqualValues(len(wire), *chained.mbufA.PktLen())
	assert.EqualValues(2, *chained.mbufA.NbSegs())
	assert.EqualValues(0, *chained.mbufA.DataLen())
}
