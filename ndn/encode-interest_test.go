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
	tpl.Encode(pkt, nil, 0xA0A1A2A3, nil)
	encoded := pkt.ReadAll()
	assert.Equal(dpdktestenv.BytesFromHex("050E name=0706080141080142 "+
		"nonce=0A04A3A2A1A0"), encoded)

	tpl.SetCanBePrefix(true)
	tpl.SetMustBeFresh(true)
	tpl.SetInterestLifetime(9000 * time.Millisecond)
	tpl.SetHopLimit(125)

	suffix, e := ndn.ParseName("/C/D")
	require.NoError(e)

	pkt = dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	defer pkt.Close()
	tpl.Encode(pkt, suffix, 0xA0A1A2A3, nil)
	encoded = pkt.ReadAll()
	assert.Equal(dpdktestenv.BytesFromHex("0521 name=070C080141080142080143080144 "+
		"canbeprefix=2100 mustbefresh=1200 nonce=0A04A3A2A1A0 "+
		"lifetime=0C0400002328 hoplimit=22017D"), encoded)
}
