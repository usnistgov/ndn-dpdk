package tgconsumer_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/app/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestConsumer(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	nameA := ndn.ParseName("/A")
	nameB := ndn.ParseName("/B")
	cfg := tgconsumer.Config{
		Patterns: []tgconsumer.Pattern{
			{
				Weight:           50,
				Prefix:           nameA,
				CanBePrefix:      true,
				MustBeFresh:      true,
				InterestLifetime: 500,
				HopLimit:         10,
			},
			{
				Weight: 45,
				Prefix: nameB,
			},
			{
				Weight:       5,
				Prefix:       nameB,
				SeqNumOffset: 100,
			},
		},
		Interval: nnduration.Nanoseconds(200000),
	}

	consumer, e := tgconsumer.New(face.D, cfg)
	require.NoError(e)
	defer consumer.Close()
	consumer.SetLCores(tgtestenv.WorkerLCores[0], tgtestenv.WorkerLCores[1])
	tgtestenv.DemuxD.SetDest(0, consumer.RxQueue())

	nInterestsA := 0
	nInterestsB1 := 0
	nInterestsB2 := 0
	nInterestsB2Far := 0
	var lastSeqB uint64
	go func() {
		for packet := range face.Rx {
			require.NotNil(packet.Interest)
			interest := *packet.Interest
			switch {
			case nameA.IsPrefixOf(interest.Name) && len(interest.Name) == 2:
				nInterestsA++
			case nameB.IsPrefixOf(interest.Name) && len(interest.Name) == 2:
				lastComp := interest.Name.Get(-1)
				seqNum := binary.LittleEndian.Uint64(lastComp.Value)
				diff := int64(seqNum - lastSeqB)
				if nInterestsB1 == 0 || diff > -50 {
					if nInterestsB1 > 0 && nInterestsB2 > 0 {
						assert.InDelta(lastSeqB+1, seqNum, 10)
					}
					lastSeqB = seqNum
					nInterestsB1++
				} else {
					if nInterestsB1 > 0 && nInterestsB2 > 0 {
						// diff should be around -100
						if diff < -110 || diff > -90 {
							nInterestsB2Far++
						}
					}
					nInterestsB2++
				}
			default:
				assert.Fail("unexpected Interest", "%v", interest)
			}

			face.Tx <- ndn.MakeData(interest)
		}
		close(face.Tx)
	}()

	assert.InDelta(200*time.Microsecond, consumer.Interval(), float64(1*time.Microsecond))
	consumer.Launch()

	time.Sleep(900 * time.Millisecond)

	timeBeforeStop := time.Now()
	e = consumer.Stop(200 * time.Millisecond)
	assert.NoError(e)
	assert.InDelta(200*time.Millisecond, time.Since(timeBeforeStop), float64(50*time.Millisecond))

	nInterests := float64(nInterestsA + nInterestsB1 + nInterestsB2)
	assert.InDelta(4500, nInterests, 1000)
	assert.InDelta(nInterests*0.50, nInterestsA, 100)
	assert.InDelta(nInterests*0.45, nInterestsB1, 100)
	assert.InDelta(nInterests*0.05, nInterestsB2, 100)
	assert.LessOrEqual(nInterestsB2Far, nInterestsB2/10)

	cnt := consumer.ReadCounters()
	assert.InDelta(nInterests, cnt.NInterests, 500)
	assert.InDelta(nInterests, cnt.NData, 500)
	require.Len(cnt.PerPattern, 3)
	assert.InDelta(nInterestsA, cnt.PerPattern[0].NData, 100)
	assert.InDelta(nInterestsB1, cnt.PerPattern[1].NData, 100)
	assert.InDelta(nInterestsB2, cnt.PerPattern[2].NData, 100)
}
