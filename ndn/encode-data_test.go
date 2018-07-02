package ndn_test

import (
	"testing"
	"time"

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
