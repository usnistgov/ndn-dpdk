package ndn_test

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestEncodeInterest(t *testing.T) {
	assert, require := makeAR(t)

	tpl := ndn.NewInterestTemplate()
	prefix, e := ndn.ParseName("/A/B")
	require.NoError(e)
	tpl.SetNamePrefix(prefix)

	pkt := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	defer pkt.Close()
	tpl.Encode(pkt, nil, nil)
	encoded := pkt.ReadAll()
	require.Len(encoded, 16)
	assert.Equal(dpdktestenv.BytesFromHex("050E name=0706080141080142 nonce=0A04"),
		encoded[:12])

	tpl.SetCanBePrefix(true)
	tpl.SetMustBeFresh(true)
	tpl.SetInterestLifetime(9000 * time.Millisecond)
	tpl.SetHopLimit(125)

	suffix, e := ndn.ParseName("/C/D")
	require.NoError(e)

	pkt = dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	defer pkt.Close()
	tpl.Encode(pkt, suffix, nil)
	encoded = pkt.ReadAll()
	require.Len(encoded, 35)
	assert.Equal(dpdktestenv.BytesFromHex("0521 name=070C080141080142080143080144 "+
		"canbeprefix=2100 mustbefresh=1200 nonce=0A04"), encoded[:22])
	assert.Equal(dpdktestenv.BytesFromHex("lifetime=0C0400002328 hoplimit=22017D"),
		encoded[26:])
}
