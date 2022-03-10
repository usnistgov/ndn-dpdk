package tgconsumer_test

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgtestenv"
	"github.com/usnistgov/ndn-dpdk/app/tgconsumer"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestConsumer(t *testing.T) {
	assert, require := makeAR(t)

	face := intface.MustNew()
	defer face.D.Close()

	nameA, nameB, nameC := ndn.ParseName("/A"), ndn.ParseName("/B"), ndn.ParseName("/C")
	contentC := make([]byte, 100)
	cfg := tgconsumer.Config{
		Interval: nnduration.Nanoseconds(200 * time.Microsecond),
		Patterns: []tgconsumer.Pattern{
			{
				Weight: 30,
				InterestTemplateConfig: ndni.InterestTemplateConfig{
					Prefix:           nameA,
					CanBePrefix:      true,
					MustBeFresh:      true,
					InterestLifetime: 500,
					HopLimit:         10,
				},
			},
			{
				Weight: 45,
				InterestTemplateConfig: ndni.InterestTemplateConfig{
					Prefix: nameB,
				},
			},
			{
				Weight: 5,
				InterestTemplateConfig: ndni.InterestTemplateConfig{
					Prefix: nameB,
				},
				SeqNumOffset: 100,
			},
			{
				Weight: 20,
				InterestTemplateConfig: ndni.InterestTemplateConfig{
					Prefix: nameC,
				},
				Digest: &ndni.DataGenConfig{
					PayloadLen: len(contentC),
				},
			},
		},
	}

	c, e := tgconsumer.New(face.D, cfg)
	require.NoError(e)
	defer c.Close()
	tgtestenv.Open(t, c)

	nInterestsA, nInterestsB1, nInterestsB2, nInterestsB2Far, nInterestsC := 0, 0, 0, 0, 0
	var lastSeqB uint64
	go func() {
		for packet := range face.Rx {
			require.NotNil(packet.Interest)
			interest := *packet.Interest
			switch {
			case nameA.IsPrefixOf(interest.Name) && len(interest.Name) == 2:
				nInterestsA++
				face.Tx <- ndn.MakeData(interest)
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
				face.Tx <- ndn.MakeData(interest)
			case nameC.IsPrefixOf(interest.Name) && len(interest.Name) == 3 && interest.Name[2].Type == an.TtImplicitSha256DigestComponent:
				nInterestsC++
				face.Tx <- ndn.MakeData(interest.Name.GetPrefix(-1), contentC, packet.Lp)
			default:
				assert.Fail("unexpected Interest", "%v", interest)
			}
		}
		close(face.Tx)
	}()

	assert.InDelta(200*time.Microsecond, c.Interval(), float64(1*time.Microsecond))
	c.Launch()

	time.Sleep(900 * time.Millisecond)

	timeBeforeStop := time.Now()
	e = c.StopDelay(200 * time.Millisecond)
	assert.NoError(e)
	assert.InDelta(200*time.Millisecond, time.Since(timeBeforeStop), float64(50*time.Millisecond))

	nInterests := float64(nInterestsA + nInterestsB1 + nInterestsB2 + nInterestsC)
	assert.InDelta(4500, nInterests, 1000)
	assert.InDelta(nInterests*0.30, nInterestsA, 100)
	assert.InDelta(nInterests*0.45, nInterestsB1, 100)
	assert.InDelta(nInterests*0.05, nInterestsB2, 100)
	assert.InDelta(nInterests*0.20, nInterestsC, 100)
	assert.LessOrEqual(nInterestsB2Far, nInterestsB2/10)

	cnt := c.Counters()
	assert.InDelta(nInterests, cnt.NInterests, 500)
	assert.InDelta(nInterests, cnt.NData, 500)
	require.Len(cnt.PerPattern, 4)
	assert.InDelta(nInterestsA, cnt.PerPattern[0].NData, 100)
	assert.InDelta(nInterestsB1, cnt.PerPattern[1].NData, 100)
	assert.InDelta(nInterestsB2, cnt.PerPattern[2].NData, 100)
	assert.InDelta(nInterestsC, cnt.PerPattern[3].NData, 100)
}
