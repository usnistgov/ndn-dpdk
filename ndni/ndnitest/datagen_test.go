package ndnitest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func testDataGen(t *testing.T, linearize bool) {
	assert, require := makeAR(t)

	tplMbuf := ndnitestenv.Payload.Alloc()
	dataInput := ndn.MakeData(ndn.ParseName("/suffix"), ndn.ContentType(an.ContentLink), 3016*time.Millisecond, []byte{0xC0, 0xC1})

	var gen ndni.DataGen
	gen.Init(tplMbuf, dataInput, linearize)
	defer gen.Close()

	var mp ndni.Mempools
	mp.Assign(eal.NumaSocket{}, ndni.DataMempool)
	pkt := gen.Encode(ndn.ParseName("/prefix"), &mp)
	require.NotNil(pkt)
	assert.Equal(!linearize, pkt.Mbuf().IsSegmented())

	data := pkt.ToNPacket().Data
	require.NotNil(data)
	nameEqual(assert, "/prefix/suffix", data)
	assert.EqualValues(an.ContentLink, data.ContentType)
	assert.Equal(3016*time.Millisecond, data.Freshness)
	assert.Equal([]byte{0xC0, 0xC1}, data.Content)
}

func TestDataGen(t *testing.T) {
	testDataGen(t, false)
	testDataGen(t, true)
}
