package ndn_test

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestenv"
)

func TestEncodeData(t *testing.T) {
	assert, require := makeAR(t)

	m := ndntestenv.Packet.Alloc()
	defer m.Close()

	namePrefix, e := ndn.ParseName("/A/B")
	require.NoError(e)
	nameSuffix, e := ndn.ParseName("/C")
	require.NoError(e)
	freshnessPeriod := 11742 * time.Millisecond
	content := ndn.TlvBytes{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7}

	ndn.EncodeData(m, namePrefix, nameSuffix, freshnessPeriod, content)
	pkt := ndn.PacketFromMbuf(m)
	e = pkt.ParseL3(ndntestenv.Name.Pool())
	require.NoError(e)
	data := pkt.AsData()

	ndntestenv.NameEqual(assert, "/A/B/C", data)
	assert.Equal(freshnessPeriod, data.GetFreshnessPeriod())
}

func TestDataGen(t *testing.T) {
	assert, require := makeAR(t)

	mbufs := ndntestenv.Packet.Pool().MustAlloc(2)
	mi := mbuftestenv.Indirect.Alloc()

	namePrefix, e := ndn.ParseName("/A/B")
	require.NoError(e)
	nameSuffix, e := ndn.ParseName("/C")
	require.NoError(e)
	freshnessPeriod := 11742 * time.Millisecond
	content := ndn.TlvBytes{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7}

	gen := ndn.NewDataGen(mbufs[1], nameSuffix, freshnessPeriod, content)
	defer gen.Close()
	gen.Encode(mbufs[0], mi, namePrefix)

	pkt := ndn.PacketFromMbuf(mbufs[0])
	defer mbufs[0].Close()
	e = pkt.ParseL3(ndntestenv.Name.Pool())
	require.NoError(e)
	data := pkt.AsData()

	ndntestenv.NameEqual(assert, "/A/B/C", data)
	assert.Equal(freshnessPeriod, data.GetFreshnessPeriod())
}
