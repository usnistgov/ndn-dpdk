package pingclient_test

import (
	"encoding/binary"
	"testing"
	"time"

	"ndn-dpdk/app/ping/pingtestenv"
	"ndn-dpdk/app/pingclient"
	"ndn-dpdk/core/nnduration"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestClient(t *testing.T) {
	assert, require := makeAR(t)

	face := pingtestenv.MakeMockFace()
	defer face.Close()
	face.DisableTxRecorders()

	nameA := ndn.MustParseName("/A")
	nameB := ndn.MustParseName("/B")
	cfg := pingclient.Config{
		Patterns: []pingclient.Pattern{
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

	client, e := pingclient.New(face, cfg)
	require.NoError(e)
	defer client.Close()
	client.SetLCores(pingtestenv.SlaveLCores[0], pingtestenv.SlaveLCores[1])

	rx := pingtestenv.MakeRxFunc(client.GetRxQueue())
	nInterestsA := 0
	nInterestsB1 := 0
	nInterestsB2 := 0
	var lastSeqB uint64
	face.OnTxInterest(func(interest *ndn.Interest) {
		interestName := interest.GetName()
		switch {
		case interestName.Compare(nameA) == ndn.NAMECMP_RPREFIX && interestName.Len() == 2:
			nInterestsA++
		case interestName.Compare(nameB) == ndn.NAMECMP_RPREFIX && interestName.Len() == 2:
			lastComp := interestName.GetComp(interestName.Len() - 1)
			seqnum := binary.LittleEndian.Uint64(lastComp.GetValue())
			diff := int64(seqnum - lastSeqB)
			if nInterestsB1 == 0 || diff > -50 {
				if nInterestsB1 > 0 && nInterestsB2 > 0 {
					assert.InDelta(lastSeqB+1, seqnum, 10)
				}
				lastSeqB = seqnum
				nInterestsB1++
			} else {
				if nInterestsB1 > 0 && nInterestsB2 > 0 {
					assert.InDelta(lastSeqB-100, seqnum, 10)
				}
				nInterestsB2++
			}
		default:
			assert.Fail("unexpected Interest", "%s", interest)
		}
		data := ndntestutil.MakeData(interest.GetName().String())
		ndntestutil.CopyPitToken(data, interest)
		rx(data)
	})

	assert.InDelta(200*time.Microsecond, client.GetInterval(), float64(1*time.Microsecond))
	client.Launch()

	time.Sleep(900 * time.Millisecond)

	timeBeforeStop := time.Now()
	e = client.Stop(200 * time.Millisecond)
	assert.NoError(e)
	assert.InDelta(200*time.Millisecond, time.Since(timeBeforeStop), float64(50*time.Millisecond))

	nInterests := float64(nInterestsA + nInterestsB1 + nInterestsB2)
	assert.InDelta(4500, nInterests, 1000)
	assert.InDelta(nInterests*0.50, nInterestsA, 100)
	assert.InDelta(nInterests*0.45, nInterestsB1, 100)
	assert.InDelta(nInterests*0.05, nInterestsB2, 100)

	cnt := client.ReadCounters()
	assert.InDelta(nInterests, cnt.NInterests, 100)
	assert.InDelta(nInterests, cnt.NData, 100)
	require.Len(cnt.PerPattern, 3)
	assert.InDelta(nInterestsA, cnt.PerPattern[0].NData, 100)
	assert.InDelta(nInterestsB1, cnt.PerPattern[1].NData, 100)
	assert.InDelta(nInterestsB2, cnt.PerPattern[2].NData, 100)
}
