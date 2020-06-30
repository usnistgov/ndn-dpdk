package ndn_test

import (
	"bytes"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestLpFragmenter(t *testing.T) {
	assert, require := makeAR(t)

	data := ndn.MakeData("/D", bytes.Repeat([]byte{0xCC}, 3000))
	packet := data.ToPacket()
	packet.Lp.PitToken = ndn.PitTokenFromUint(0x808ecd3df4e1b062)

	fragmenter := ndn.NewLpFragmenter(1000)
	frags, e := fragmenter.Fragment(packet)
	require.NoError(e)
	require.Len(frags, 4)

	for _, frag := range frags {
		wire, _ := tlv.Encode(frag)
		assert.LessOrEqual(len(wire), 1000)
	}

	tooSmall := ndn.NewLpFragmenter(10)
	_, e = tooSmall.Fragment(packet)
	assert.Error(e)
}
