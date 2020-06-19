package ndni_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestEncodeData(t *testing.T) {
	assert, require := makeAR(t)

	m := ndnitestenv.Packet.Alloc()
	defer m.Close()

	prefix := ndn.ParseName("/A/B")
	suffix := ndn.ParseName("/C")
	freshnessPeriod := 11742 * time.Millisecond
	content := []byte{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7}

	ndni.EncodeData(m, prefix, suffix, freshnessPeriod, content)
	pkt := ndni.PacketFromMbuf(m)
	e := pkt.ParseL3(ndnitestenv.Name.Pool())
	require.NoError(e)
	data := pkt.AsData()

	ndntestenv.NameEqual(assert, "/A/B/C", data)
	assert.Equal(freshnessPeriod, data.GetFreshnessPeriod())
}

func TestDataGen(t *testing.T) {
	assert, require := makeAR(t)

	mbufs := ndnitestenv.Packet.Pool().MustAlloc(2)
	mi := mbuftestenv.Indirect.Alloc()

	prefix := ndn.ParseName("/A/B")
	suffix := ndn.ParseName("/C")
	freshnessPeriod := 11742 * time.Millisecond
	content := []byte{0xC0, 0xC1, 0xC2, 0xC3, 0xC4, 0xC5, 0xC6, 0xC7}

	gen := ndni.NewDataGen(mbufs[1], suffix, freshnessPeriod, content)
	defer gen.Close()
	gen.Encode(mbufs[0], mi, prefix)

	pkt := ndni.PacketFromMbuf(mbufs[0])
	defer mbufs[0].Close()
	e := pkt.ParseL3(ndnitestenv.Name.Pool())
	require.NoError(e)
	data := pkt.AsData()

	ndntestenv.NameEqual(assert, "/A/B/C", data)
	assert.Equal(freshnessPeriod, data.GetFreshnessPeriod())
}
