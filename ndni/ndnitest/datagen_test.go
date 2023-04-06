package ndnitest

import (
	"bytes"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestDataGen(t *testing.T) {
	payloadMp := ndni.PayloadMempool.Get(eal.NumaSocket{})
	tplMbuf := payloadMp.MustAlloc(1)[0]
	content := bytes.Repeat([]byte{0xC0, 0xC1, 0xC2, 0xC3}, 300)

	var gen ndni.DataGen
	gen.Init(tplMbuf, ndn.ParseName("/suffix"), ndn.ContentType(an.ContentLink), 3016*time.Millisecond, content)
	defer gen.Close()

	var mp ndni.Mempools
	mp.Assign(eal.NumaSocket{}, ndni.DataMempool)

	checkData := func(t testing.TB, pkt *ndni.Packet) {
		assert, require := makeAR(t)
		require.NotNil(pkt)

		data := pkt.ToNPacket().Data
		require.NotNil(data)
		nameEqual(assert, "/prefix/suffix", data)
		assert.EqualValues(an.ContentLink, data.ContentType)
		assert.Equal(3016*time.Millisecond, data.Freshness)
		assert.Equal(content, data.Content)
	}

	t.Run("chained", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := gen.Encode(ndn.ParseName("/prefix"), &mp, 0)
		checkData(t, pkt)

		m := pkt.Mbuf()
		if segs := m.SegmentBytes(); assert.Len(segs, 3) {
			assert.Less(len(segs[0]), 500)
			assert.Len(segs[2], ndni.DataEncNullSigLen)
		}
	})

	t.Run("1800", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := gen.Encode(ndn.ParseName("/prefix"), &mp, 1800)
		checkData(t, pkt)

		m := pkt.Mbuf()
		if segs := m.SegmentBytes(); assert.Len(segs, 1) {
			assert.Less(len(segs[0]), 1800)
		}
	})

	t.Run("1000", func(t *testing.T) {
		assert, _ := makeAR(t)
		pkt := gen.Encode(ndn.ParseName("/prefix"), &mp, 1000)
		checkData(t, pkt)

		m := pkt.Mbuf()
		if segs := m.SegmentBytes(); assert.Len(segs, 2) {
			assert.Greater(len(segs[0]), 500)
		}
	})
}
