package ndnitest

/*
#include "../../csrc/ndni/packet.h"
*/
import "C"
import (
	"bytes"
	"testing"
	"unsafe"

	"github.com/pkg/math"
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

		if tt.Bad {
			assert.False(ok, tt.Input)
		} else if assert.True(ok, tt.Input) {
			assert.EqualValues(tt.SeqNum, C.LpL2_GetSeqNum(&lph.l2), tt.Input)
			assert.EqualValues(tt.FragIndex, lph.l2.fragIndex, tt.Input)
			assert.EqualValues(math.MaxUint16(1, tt.FragCount), lph.l2.fragCount, tt.Input)
			if len(tt.PitToken) == 0 {
				assert.Zero(lph.l3.pitToken.length, tt.Input)
			} else {
				assert.Equal(tt.PitToken, cptr.AsByteSlice(&lph.l3.pitToken.value)[:lph.l3.pitToken.length], tt.Input)
			}
			assert.EqualValues(tt.NackReason, lph.l3.nackReason, tt.Input)
			assert.EqualValues(tt.CongMark, lph.l3.congMark, tt.Input)
			assert.EqualValues(tt.PayloadL, p.Len(), tt.Input)
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
	assert.EqualValues(len(wire), single.mbuf.pkt_len)
	assert.EqualValues(1, single.mbuf.nb_segs)
	assert.EqualValues(len(wire), single.mbuf.data_len)

	linearFrag := toPacket(unsafe.Pointer(p.N.Clone(mp, ndni.PacketTxAlign{
		Linearize:           true,
		FragmentPayloadSize: 500,
	})))
	require.NotNil(linearFrag)
	defer linearFrag.Close()
	assert.EqualValues(len(wire), linearFrag.mbuf.pkt_len)
	assert.EqualValues(3, linearFrag.mbuf.nb_segs)
	assert.EqualValues(500, linearFrag.mbuf.data_len)

	chained := toPacket(unsafe.Pointer(p.N.Clone(mp, ndni.PacketTxAlign{
		Linearize: false,
	})))
	require.NotNil(chained)
	defer chained.Close()
	assert.EqualValues(len(wire), chained.mbuf.pkt_len)
	assert.EqualValues(2, chained.mbuf.nb_segs)
	assert.EqualValues(0, chained.mbuf.data_len)
}
