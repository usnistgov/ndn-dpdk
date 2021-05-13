package tgproducer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/app/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
)

func TestPatterns(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	p, e := tgproducer.New(face.D, iface.PktQueueConfig{}, 1)
	require.NoError(e)
	defer p.Close()
	require.NoError(ealthread.DefaultAllocator.AllocThread(p.Workers()...))
	p.ConnectRxQueues(tgtestenv.DemuxI)

	nameA := ndn.ParseName("/A")
	nameB := ndn.ParseName("/B")
	require.NoError(p.SetPatterns([]tgproducer.Pattern{
		{
			Prefix: nameA,
			Replies: []tgproducer.Reply{
				{
					Weight:          60,
					FreshnessPeriod: 100,
					PayloadLen:      1000,
				},
				{
					Weight:          40,
					Suffix:          ndn.ParseName("/Z"),
					FreshnessPeriod: 100,
					PayloadLen:      2000,
				},
			},
		},
		{
			Prefix: nameB,
			Replies: []tgproducer.Reply{
				{
					Nack: an.NackCongestion,
				},
			},
		},
		{
			Prefix: ndn.ParseName("/C"),
			Replies: []tgproducer.Reply{
				{
					Timeout: true,
				},
			},
		},
	}))

	nDataA0 := 0
	nDataA1 := 0
	nNacksB := 0
	go func() {
		for packet := range face.Rx {
			switch {
			case packet.Data != nil:
				dataName := packet.Data.Name
				switch {
				case nameA.IsPrefixOf(dataName) && len(dataName) == 2:
					nDataA0++
				case nameA.IsPrefixOf(dataName) && len(dataName) == 3:
					nDataA1++
				default:
					assert.Fail("unexpected Data", "%v", *packet.Data)
				}
			case packet.Nack != nil:
				interestName := packet.Nack.Interest.Name
				switch {
				case nameB.IsPrefixOf(interestName) && len(interestName) == 2:
					nNacksB++
				default:
					assert.Fail("unexpected Nack", "%v", *packet.Nack)
				}
			default:
				assert.Fail("unexpected packet")
			}
		}
	}()

	p.Launch()

	func() {
		for i := 0; i < 100; i++ {
			face.Tx <- ndn.MakeInterest(fmt.Sprintf("/A/%d", i))
			face.Tx <- ndn.MakeInterest(fmt.Sprintf("/B/%d", i))
			face.Tx <- ndn.MakeInterest(fmt.Sprintf("/C/%d", i))
			time.Sleep(50 * time.Microsecond)
		}
		time.Sleep(200 * time.Millisecond)
		close(face.Tx)
	}()

	e = p.Stop()
	assert.NoError(e)
	assert.Equal(100, nDataA0+nDataA1)
	assert.InDelta(60, nDataA0, 20)
	assert.InDelta(40, nDataA1, 20)
	assert.Equal(100, nNacksB)
}

func TestDataProducer(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	p, e := tgproducer.New(face.D, iface.PktQueueConfig{}, 2)
	require.NoError(e)
	defer p.Close()
	require.NoError(ealthread.DefaultAllocator.AllocThread(p.Workers()...))
	p.ConnectRxQueues(tgtestenv.DemuxI)

	nameP := ndn.ParseName("/P")
	require.NoError(p.SetPatterns([]tgproducer.Pattern{
		{
			Prefix: nameP,
			Replies: []tgproducer.Reply{
				{
					FreshnessPeriod: 100,
					PayloadLen:      1000,
				},
			},
		},
	}))

	p.Launch()

	fw := l3.NewForwarder()
	fwFace, e := fw.AddFace(face.A)
	require.NoError(e)
	fwFace.AddRoute(ndn.Name{})

	fetcher := segmented.Fetch(nameP, segmented.FetchOptions{
		Fw:           fw,
		SegmentBegin: 0,
		SegmentEnd:   5000,
		RetxLimit:    10,
	})

	ordered := make(chan *ndn.Data)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		e := fetcher.Ordered(ctx, ordered)
		assert.NoError(e)
	}()

	received := 0
	for range ordered {
		received++
	}
	assert.Equal(5000, received)

	e = p.Stop()
	assert.NoError(e)
}
