package rdr_test

import (
	"context"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/endpoint"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/rdr"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

var (
	makeAR    = testenv.MakeAR
	nameEqual = ndntestenv.NameEqual
)

func TestDiscoveryInterest(t *testing.T) {
	assert, _ := makeAR(t)

	diA := rdr.MakeDiscoveryInterest(ndn.ParseName("/A"))
	nameEqual(assert, "/A/32=metadata", diA)
	assert.True(diA.CanBePrefix)
	assert.True(diA.MustBeFresh)
	assert.True(rdr.IsDiscoveryInterest(diA))

	diB := rdr.MakeDiscoveryInterest(ndn.ParseName("/B/32=metadata"))
	nameEqual(assert, "/B/32=metadata", diB)
	assert.True(rdr.IsDiscoveryInterest(diB))

	assert.False(rdr.IsDiscoveryInterest(ndn.MakeInterest(
		"/C",
		ndn.CanBePrefixFlag,
		ndn.MustBeFreshFlag,
	)))
	assert.False(rdr.IsDiscoveryInterest(ndn.MakeInterest(
		"/C/32=metadata",
		ndn.MustBeFreshFlag,
	)))
	assert.False(rdr.IsDiscoveryInterest(ndn.MakeInterest(
		"/C/32=metadata",
		ndn.CanBePrefixFlag,
	)))
}

func TestRetrieve(t *testing.T) {
	t.Cleanup(l3.DeleteDefaultForwarder)
	assert, require := makeAR(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p0, e := endpoint.Produce(ctx, endpoint.ProducerOptions{
		Prefix: ndn.ParseName("/P0"),
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			assert.True(rdr.IsDiscoveryInterest(interest))
			return ndn.MakeData(interest, ndn.ContentType(an.ContentNack)), nil
		},
	})
	require.NoError(e)
	defer p0.Close()

	p1, e := endpoint.Produce(ctx, endpoint.ProducerOptions{
		Prefix: ndn.ParseName("/P1"),
		Handler: func(ctx context.Context, interest ndn.Interest) (ndn.Data, error) {
			assert.True(rdr.IsDiscoveryInterest(interest))
			var m rdr.Metadata
			m.Name = ndn.ParseName("/B/7")
			wire, e := m.MarshalBinary()
			return ndn.MakeData(
				interest.Name.Append(ndn.NameComponentFrom(an.TtVersionNameComponent, tlv.NNI(time.Now().UnixMicro()))),
				time.Millisecond,
				wire,
			), e
		},
	})
	require.NoError(e)
	defer p1.Close()

	var m0, m1 rdr.Metadata
	e0 := rdr.RetrieveMetadata(ctx, &m0, ndn.ParseName("/P0"), endpoint.ConsumerOptions{})
	assert.ErrorIs(e0, ndn.ErrContentType)
	e1 := rdr.RetrieveMetadata(ctx, &m1, ndn.ParseName("/P1"), endpoint.ConsumerOptions{})
	if assert.NoError(e1) {
		nameEqual(assert, "/B/7", m1)
	}
}
