package ndni_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestDataDecode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		bad       bool
		name      string
		freshness int
	}{
		{input: "0600", bad: true},                        // missing Name
		{input: "0604 meta=1400 content=1500", bad: true}, // missing Name
		{input: "0602 name=0700", name: "/"},
		{input: "0605 name=0703080141", name: "/A"},
		{input: "0615 name=0703080142 meta=140C (180102 fp=190201FF 1A03080142) content=1500", name: "/B", freshness: 0x01FF},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsMbuf().Close()
		e := pkt.ParseL3(ndnitestenv.Name.Pool())
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			if !assert.Equal(ndni.L3PktTypeData, pkt.L3Type(), tt.input) {
				continue
			}
			data := pkt.AsData()
			assert.Implements((*ndni.IL3Packet)(nil), data)
			ndntestenv.NameEqual(assert, tt.name, data, tt.input)
			assert.EqualValues(tt.freshness, data.FreshnessPeriod()/time.Millisecond, tt.input)
		}
	}
}

func TestDataSatisfy(t *testing.T) {
	assert, _ := makeAR(t)

	interestExact := makeInterest("/B")
	interestPrefix := makeInterest("/B", ndn.CanBePrefixFlag)
	interestFresh := makeInterest("/B", ndn.MustBeFreshFlag)

	tests := []struct {
		data        *ndni.Data
		exactMatch  ndni.DataSatisfyResult
		prefixMatch ndni.DataSatisfyResult
		freshMatch  ndni.DataSatisfyResult
	}{
		{makeData("/A", time.Second),
			ndni.DataSatisfyNo, ndni.DataSatisfyNo, ndni.DataSatisfyNo},
		{makeData("/2=B", time.Second),
			ndni.DataSatisfyNo, ndni.DataSatisfyNo, ndni.DataSatisfyNo},
		{makeData("/B", time.Second),
			ndni.DataSatisfyYes, ndni.DataSatisfyYes, ndni.DataSatisfyYes},
		{makeData("/B", time.Duration(0)),
			ndni.DataSatisfyYes, ndni.DataSatisfyYes, ndni.DataSatisfyNo},
		{makeData("/B/0", time.Second),
			ndni.DataSatisfyNo, ndni.DataSatisfyYes, ndni.DataSatisfyNo},
		{makeData("/", time.Second),
			ndni.DataSatisfyNo, ndni.DataSatisfyNo, ndni.DataSatisfyNo},
		{makeData("/C", time.Second),
			ndni.DataSatisfyNo, ndni.DataSatisfyNo, ndni.DataSatisfyNo},
	}
	for i, tt := range tests {
		assert.Equal(tt.exactMatch, tt.data.CanSatisfy(*interestExact), "%d", i)
		assert.Equal(tt.prefixMatch, tt.data.CanSatisfy(*interestPrefix), "%d", i)
		assert.Equal(tt.freshMatch, tt.data.CanSatisfy(*interestFresh), "%d", i)

		if tt.exactMatch == ndni.DataSatisfyYes {
			interestImplicit := makeInterest(tt.data.ToNData().FullName().String())
			assert.Equal(ndni.DataSatisfyNeedDigest, tt.data.CanSatisfy(*interestImplicit))
			tt.data.ComputeImplicitDigest()
			assert.Equal(ndni.DataSatisfyYes, tt.data.CanSatisfy(*interestImplicit))
			ndnitestenv.ClosePacket(interestImplicit)
		}

		ndnitestenv.ClosePacket(tt.data)
	}

	ndnitestenv.ClosePacket(interestExact)
	ndnitestenv.ClosePacket(interestPrefix)
	ndnitestenv.ClosePacket(interestFresh)
}

func TestDataDigest(t *testing.T) {
	assert, require := makeAR(t)

	cd, e := cryptodev.MultiSegDrv.Create(cryptodev.Config{}, eal.NumaSocket{})
	require.NoError(e)
	defer cd.Close()
	qp := cd.QueuePair(0)
	mp, e := cryptodev.NewOpPool(cryptodev.OpPoolConfig{}, eal.NumaSocket{})
	require.NoError(e)
	defer mp.Close()

	names := []string{
		"/",
		"/A",
		"/B",
		"/C",
	}
	inputs := make([]*ndni.Data, 4)
	for i, name := range names {
		inputs[i] = makeData(name)
	}

	ops, e := mp.Alloc(cryptodev.OpSymmetric, 4)
	require.NoError(e)
	for i, data := range inputs {
		assert.Nil(data.CachedImplicitDigest())
		data.DigestPrepare(ops[i])
	}
	assert.Equal(4, qp.EnqueueBurst(ops))

	assert.Equal(4, qp.DequeueBurst(ops))
	for i, op := range ops {
		data, e := ndni.DataDigestFinish(op)
		assert.NoError(e)
		if assert.NotNil(data) {
			ndntestenv.NameEqual(assert, names[i], data)
			assert.Equal(data.ToNData().ComputeDigest(), data.CachedImplicitDigest())
		}
	}
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
	assert.Equal(freshnessPeriod, data.FreshnessPeriod())
}
