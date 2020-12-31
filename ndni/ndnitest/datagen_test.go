package ndnitest

import (
	"bytes"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func testDataGen(t *testing.T, fragmentPayloadSize int, checkMbuf func(m *pktmbuf.Packet)) {
	assert, require := makeAR(t)
	payloadMp := ndni.PayloadMempool.Get(eal.NumaSocket{})

	tplMbuf := payloadMp.MustAlloc(1)[0]
	content := bytes.Repeat([]byte{0xC0, 0xC1, 0xC2, 0xC3}, 300)
	dataInput := ndn.MakeData(ndn.ParseName("/suffix"), ndn.ContentType(an.ContentLink), 3016*time.Millisecond, content)

	var gen ndni.DataGen
	gen.Init(tplMbuf, dataInput)
	defer gen.Close()

	var mp ndni.Mempools
	mp.Assign(eal.NumaSocket{}, ndni.DataMempool)
	pkt := gen.Encode(ndn.ParseName("/prefix"), &mp, fragmentPayloadSize)
	require.NotNil(pkt)

	data := pkt.ToNPacket().Data
	require.NotNil(data)
	nameEqual(assert, "/prefix/suffix", data)
	assert.EqualValues(an.ContentLink, data.ContentType)
	assert.Equal(3016*time.Millisecond, data.Freshness)
	assert.Equal(content, data.Content)
}

func TestDataGen(t *testing.T) {
	assert, _ := makeAR(t)

	testDataGen(t, 0, func(m *pktmbuf.Packet) {
		segs := m.SegmentLengths()
		assert.Len(segs, 2)
		assert.Less(segs[0], 500)
	})

	testDataGen(t, 3000, func(m *pktmbuf.Packet) {
		assert.Len(m.SegmentLengths(), 1)
	})

	testDataGen(t, 1000, func(m *pktmbuf.Packet) {
		segs := m.SegmentLengths()
		assert.Len(segs, 2)
		assert.Greater(segs[0], 500)
	})
}
