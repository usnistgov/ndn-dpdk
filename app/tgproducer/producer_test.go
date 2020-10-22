package tgproducer_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/tgproducer"
	"github.com/usnistgov/ndn-dpdk/app/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func TestProducer(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	nameA := ndn.ParseName("/A")
	nameB := ndn.ParseName("/B")
	cfg := tgproducer.Config{
		Patterns: []tgproducer.Pattern{
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
		},
		Nack: true,
	}

	producer, e := tgproducer.New(face.D, 0, cfg)
	require.NoError(e)
	defer producer.Close()
	producer.SetLCore(tgtestenv.WorkerLCores[0])
	tgtestenv.DemuxI.SetDest(0, producer.RxQueue())

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

	producer.Launch()

	func() {
		for i := 0; i < 100; i++ {
			face.Tx <- ndn.MakeInterest(fmt.Sprintf("/A/%d", i))
			face.Tx <- ndn.MakeInterest(fmt.Sprintf("/B/%d", i))
			face.Tx <- ndn.MakeInterest(fmt.Sprintf("/C/%d", i))
			time.Sleep(50 * time.Microsecond)
		}
		time.Sleep(80 * time.Millisecond)
		close(face.Tx)
	}()

	e = producer.Stop()
	assert.NoError(e)
	assert.Equal(100, nDataA0+nDataA1)
	assert.InDelta(60, nDataA0, 20)
	assert.InDelta(40, nDataA1, 20)
	assert.Equal(100, nNacksB)
}
