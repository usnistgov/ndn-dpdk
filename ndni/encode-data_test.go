package ndni_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndntestenv"
)

func TestEncodeData(t *testing.T) {
	assert, require := makeAR(t)

	m := ndntestenv.Packet.Alloc()
	defer m.Close()

	namePrefix, e := ndni.ParseName("/A/B")
	require.NoError(e)
	nameSuffix, e := ndni.ParseName("/C")
	require.NoError(e)
	freshnessPeriod := 11742 * time.Millisecond
	content := ndni.TlvBytes{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7}

	ndni.EncodeData(m, namePrefix, nameSuffix, freshnessPeriod, content)
	pkt := ndni.PacketFromMbuf(m)
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

	namePrefix, e := ndni.ParseName("/A/B")
	require.NoError(e)
	nameSuffix, e := ndni.ParseName("/C")
	require.NoError(e)
	freshnessPeriod := 11742 * time.Millisecond
	content := ndni.TlvBytes{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7}

	gen := ndni.NewDataGen(mbufs[1], nameSuffix, freshnessPeriod, content)
	defer gen.Close()
	gen.Encode(mbufs[0], mi, namePrefix)

	pkt := ndni.PacketFromMbuf(mbufs[0])
	defer mbufs[0].Close()
	e = pkt.ParseL3(ndntestenv.Name.Pool())
	require.NoError(e)
	data := pkt.AsData()

	ndntestenv.NameEqual(assert, "/A/B/C", data)
	assert.Equal(freshnessPeriod, data.GetFreshnessPeriod())
}
