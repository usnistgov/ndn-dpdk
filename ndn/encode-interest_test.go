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
	assert.Equal(dpdktestenv.BytesFromHex("0511 name=0706080141080142 "+
		"nonce=0A04A3A2A1A0 hoplimit=2201FF"), encoded)

	tpl.SetCanBePrefix(true)
	tpl.SetMustBeFresh(true)
	fhName, e := ndn.ParseName("/E")
	require.NoError(e)
	tpl.AppendFH(15601, fhName)
	fhName, e = ndn.ParseName("/F")
	require.NoError(e)
	tpl.AppendFH(6323, fhName)
	tpl.SetInterestLifetime(9000 * time.Millisecond)
	tpl.SetHopLimit(125)

	suffix, e := ndn.ParseName("/C/D")
	require.NoError(e)

	pkt = dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	defer pkt.Close()
	tpl.Encode(pkt, suffix, 0xA0A1A2A3, nil)
	encoded = pkt.ReadAll()
	assert.Equal(dpdktestenv.BytesFromHex("053D name=070C080141080142080143080144 "+
		"canbeprefix=2100 mustbefresh=1200 "+
		"fh=1E1A(1F0B pref=1E0400003CF1 name=0703080145)(1F0B pref=1E04000018B3 name=0703080146) "+
		"nonce=0A04A3A2A1A0 lifetime=0C0400002328 hoplimit=22017D"), encoded)
}

func TestMakeInterest(t *testing.T) {
	assert, require := makeAR(t)

	m1 := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	_, e := ndn.MakeInterest(m1, "/A/B", uint32(0xA0A1A2A3))
	require.NoError(e)
	defer m1.Close()
	encoded1 := m1.AsPacket().ReadAll()
	assert.Equal(dpdktestenv.BytesFromHex("0511 name=0706080141080142 "+
		"nonce=0A04A3A2A1A0 hoplimit=2201FF"), encoded1)

	m2 := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	_, e = ndn.MakeInterest(m2, "/A/B/C/D", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag,
		ndn.FHDelegation{15601, "/E"}, ndn.FHDelegation{6323, "/F"},
		uint32(0xA0A1A2A3), 9000*time.Millisecond, uint8(125))
	require.NoError(e)
	defer m2.Close()
	encoded2 := m2.AsPacket().ReadAll()
	assert.Equal(dpdktestenv.BytesFromHex("053D name=070C080141080142080143080144 "+
		"canbeprefix=2100 mustbefresh=1200 "+
		"fh=1E1A(1F0B pref=1E0400003CF1 name=0703080145)(1F0B pref=1E04000018B3 name=0703080146) "+
		"nonce=0A04A3A2A1A0 lifetime=0C0400002328 hoplimit=22017D"), encoded2)
}
