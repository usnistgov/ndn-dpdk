package ndnitest

/*
#include "../../csrc/ndni/packet.h"
*/
import "C"
import (
	"bytes"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
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
			assert.EqualValues(tt.FragCount, lph.l2.fragCount, tt.Input)
			assert.EqualValues(tt.PitToken, lph.l3.pitToken, tt.Input)
			assert.EqualValues(tt.NackReason, lph.l3.nackReason, tt.Input)
			assert.EqualValues(tt.CongMark, lph.l3.congMark, tt.Input)
			assert.EqualValues(tt.PayloadL, p.Len(), tt.Input)
		}
	}
}

func ctestPacketClone(t *testing.T) {
	assert, require := makeAR(t)
	mp := makeMempoolsC()

	data := ndn.MakeData("/D", bytes.Repeat([]byte{0xC0}, 1200))
	wire, _ := tlv.Encode(data)
	p := makePacket(wire)
	defer p.Close()

	mSingle := C.Packet_ToMbuf(C.Packet_Clone(p.npkt, mp, C.PacketTxAlign{
		linearize:           true,
		fragmentPayloadSize: 7000,
	}))
	require.NotNil(mSingle)
	defer C.rte_pktmbuf_free(mSingle)
	assert.EqualValues(len(wire), mSingle.pkt_len)
	assert.EqualValues(1, mSingle.nb_segs)
	assert.EqualValues(len(wire), mSingle.data_len)

	mLinearFrag := C.Packet_ToMbuf(C.Packet_Clone(p.npkt, mp, C.PacketTxAlign{
		linearize:           true,
		fragmentPayloadSize: 500,
	}))
	require.NotNil(mLinearFrag)
	defer C.rte_pktmbuf_free(mLinearFrag)
	assert.EqualValues(len(wire), mLinearFrag.pkt_len)
	assert.EqualValues(3, mLinearFrag.nb_segs)
	assert.EqualValues(500, mLinearFrag.data_len)

	mChained := C.Packet_ToMbuf(C.Packet_Clone(p.npkt, mp, C.PacketTxAlign{
		linearize: false,
	}))
	require.NotNil(mChained)
	defer C.rte_pktmbuf_free(mChained)
	assert.EqualValues(len(wire), mChained.pkt_len)
	assert.EqualValues(2, mChained.nb_segs)
	assert.EqualValues(0, mChained.data_len)
}
