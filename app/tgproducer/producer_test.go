package tgproducer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/segmented"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestPatterns(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	nameA, nameB, nameC := ndn.ParseName("/A"), ndn.ParseName("/B"), ndn.ParseName("/C")
	cfg := tgproducer.Config{
		Patterns: []tgproducer.Pattern{
			{
				Prefix: nameA,
				Replies: []tgproducer.Reply{
					{
						Weight: 60,
						DataGenConfig: ndni.DataGenConfig{
							FreshnessPeriod: 100,
							PayloadLen:      1000,
						},
					},
					{
						Weight: 40,
						DataGenConfig: ndni.DataGenConfig{
							Suffix:          ndn.ParseName("/Z"),
							FreshnessPeriod: 100,
							PayloadLen:      2000,
						},
					},
				},
			},
			{
				Prefix: nameB,
				Replies: []tgproducer.Reply{
					{
						Nack: an.NackCongestion,
					},
					{
						Timeout: true,
					},
				},
			},
			{
				Prefix:  nameC,
				Replies: nil, // default Data reply
			},
		},
	}

	p, e := tgproducer.New(face.D, cfg)
	require.NoError(e)
	defer p.Close()
	tgtestenv.Open(t, p)

	nDataA0 := 0
	nDataA1 := 0
	nNacksB := 0
	nDataC := 0
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
				case nameC.IsPrefixOf(dataName) && len(dataName) == 4:
					nDataC++
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
		for i := range 100 {
			face.Tx <- ndn.MakeInterest(fmt.Sprintf("/A/%d", i))
			face.Tx <- ndn.MakeInterest(fmt.Sprintf("/B/%d", i))
			face.Tx <- func(i int) ndn.Interest {
				data := ndn.MakeData(fmt.Sprintf("/C/z/%d/z", i), 1*time.Millisecond)
				return ndn.MakeInterest(data.FullName())
			}(i)
			time.Sleep(50 * time.Microsecond)
		}
		time.Sleep(200 * time.Millisecond)
	}()

	e = p.Stop()
	assert.NoError(e)
	assert.Equal(100, nDataA0+nDataA1)
	assert.InDelta(60, nDataA0, 20)
	assert.InDelta(40, nDataA1, 20)
	assert.InDelta(50, nNacksB, 20)
	assert.Equal(100, nDataC)
}

func TestDataProducer(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	nameP := ndn.ParseName("/P")
	cfg := tgproducer.Config{
		NThreads: 2,
		Patterns: []tgproducer.Pattern{
			{
				Prefix: nameP,
				Replies: []tgproducer.Reply{
					{
						DataGenConfig: ndni.DataGenConfig{
							FreshnessPeriod: 100,
							PayloadLen:      1000,
						},
					},
				},
			},
		},
	}

	p, e := tgproducer.New(face.D, cfg)
	require.NoError(e)
	defer p.Close()
	tgtestenv.Open(t, p)
	p.Launch()

	fw := l3.NewForwarder()
	fwFace, e := fw.AddFace(face.A)
	require.NoError(e)
	fwFace.AddRoute(ndn.Name{})

	var fetchOpts segmented.FetchOptions
	fetchOpts.Fw = fw
	fetchOpts.SegmentEnd = 5000
	fetchOpts.RetxLimit = 10
	fetcher := segmented.Fetch(nameP, fetchOpts)

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
