package ndn_test

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestEncodeData(t *testing.T) {
	assert, require := makeAR(t)

	m := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	defer m.Close()

	namePrefix, e := ndn.ParseName("/A/B")
	require.NoError(e)
	nameSuffix, e := ndn.ParseName("/C")
	require.NoError(e)
	freshnessPeriod := 11742 * time.Millisecond
	content := ndn.TlvBytes{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7}

	ndn.EncodeData(m, namePrefix, nameSuffix, freshnessPeriod, content)
	pkt := ndn.PacketFromDpdk(m)
	e = pkt.ParseL3(theMp)
	require.NoError(e)
	data := pkt.AsData()

	assert.Equal("/A/B/C", data.GetName().String())
	assert.Equal(freshnessPeriod, data.GetFreshnessPeriod())
}

func TestDataGen(t *testing.T) {
	assert, require := makeAR(t)

	mbufs := make([]dpdk.Mbuf, 2)
	dpdktestenv.AllocBulk(dpdktestenv.MPID_DIRECT, mbufs)
	mi := dpdktestenv.Alloc(dpdktestenv.MPID_INDIRECT)

	namePrefix, e := ndn.ParseName("/A/B")
	require.NoError(e)
	nameSuffix, e := ndn.ParseName("/C")
	require.NoError(e)
	freshnessPeriod := 11742 * time.Millisecond
	content := ndn.TlvBytes{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7}

	gen := ndn.NewDataGen(mbufs[1], nameSuffix, freshnessPeriod, content)
	defer gen.Close()
	gen.Encode(mbufs[0], mi, namePrefix)

	pkt := ndn.PacketFromDpdk(mbufs[0])
	defer mbufs[0].Close()
	e = pkt.ParseL3(theMp)
	require.NoError(e)
	data := pkt.AsData()

	assert.Equal("/A/B/C", data.GetName().String())
	assert.Equal(freshnessPeriod, data.GetFreshnessPeriod())
}
