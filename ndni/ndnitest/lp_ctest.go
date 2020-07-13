package ndnitest

/*
#include "../../csrc/ndni/lp.h"
*/
import "C"
import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/ndntestvector"
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
			assert.EqualValues(tt.SeqNum, lph.l2.seqNum, tt.Input)
			assert.EqualValues(tt.FragIndex, lph.l2.fragIndex, tt.Input)
			assert.EqualValues(tt.FragCount, lph.l2.fragCount, tt.Input)
			assert.EqualValues(tt.PitToken, lph.l3.pitToken, tt.Input)
			assert.EqualValues(tt.NackReason, lph.l3.nackReason, tt.Input)
			assert.EqualValues(tt.CongMark, lph.l3.congMark, tt.Input)
			assert.EqualValues(tt.PayloadL, p.Len(), tt.Input)
		}
	}
}
