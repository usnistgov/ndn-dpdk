package ndn_test

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestEncodeData(t *testing.T) {
	assert, require := makeAR(t)

	name, e := ndn.NewName(TlvBytesFromHex("080141 080142"))
	require.NoError(e)

	payloadMbuf := dpdktestenv.PacketFromHex("C0C1C2C3C4C5C6C7")
	// note: payloadMbuf will be leaked if there's a fatal error below

	m1 := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	m2 := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	encoded := ndn.EncodeData(name, payloadMbuf, m1, m2)

	pkt := ndn.PacketFromDpdk(encoded)
	e = pkt.ParseL3(theMp)
	require.NoError(e)
	data := pkt.AsData()

	assert.Equal(2, data.GetName().Len())
	assert.Equal("/A/B", data.GetName().String())
}
