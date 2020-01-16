package pingserver_test

import (
	"fmt"
	"testing"
	"time"

	"ndn-dpdk/app/ping/pingtestenv"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestServer(t *testing.T) {
	assert, require := makeAR(t)

	face := pingtestenv.MakeMockFace()
	defer face.Close()
	face.DisableTxRecorders()

	nameA := ndn.MustParseName("/A")
	nameB := ndn.MustParseName("/B")
	cfg := pingserver.Config{
		Patterns: []pingserver.Pattern{
			{
				Prefix: nameA,
				Replies: []pingserver.Reply{
					{
						Weight:          60,
						FreshnessPeriod: 100,
						PayloadLen:      1000,
					},
					{
						Weight:          40,
						Suffix:          ndn.MustParseName("/Z"),
						FreshnessPeriod: 100,
						PayloadLen:      2000,
					},
				},
			},
			{
				Prefix: nameB,
				Replies: []pingserver.Reply{
					{
						Nack: ndn.NackReason_Congestion,
					},
				},
			},
			{
				Prefix: ndn.MustParseName("/C"),
				Replies: []pingserver.Reply{
					{
						Timeout: true,
					},
				},
			},
		},
		Nack: true,
	}

	nDataA0 := 0
	nDataA1 := 0
	nNacksB := 0
	face.OnTxData(func(data *ndn.Data) {
		dataName := data.GetName()
		switch {
		case dataName.Compare(nameA) == ndn.NAMECMP_RPREFIX && dataName.Len() == 2:
			nDataA0++
		case dataName.Compare(nameA) == ndn.NAMECMP_RPREFIX && dataName.Len() == 3:
			nDataA1++
		default:
			assert.Fail("unexpected Data", "%s", data)
		}
	})
	face.OnTxNack(func(nack *ndn.Nack) {
		interestName := nack.GetInterest().GetName()
		switch {
		case interestName.Compare(nameB) == ndn.NAMECMP_RPREFIX && interestName.Len() == 2:
			nNacksB++
		default:
			assert.Fail("unexpected Nack", "%s", nack)
		}
	})

	server, e := pingserver.New(face, cfg)
	require.NoError(e)
	defer server.Close()
	server.SetLCore(pingtestenv.SlaveLCores[0])

	rxQueue := pingtestenv.AttachRxQueue(server)
	defer rxQueue.Close()

	server.Launch()

	for i := 0; i < 100; i++ {
		interestA := ndntestutil.MakeInterest(fmt.Sprintf("/A/%d", i))
		interestB := ndntestutil.MakeInterest(fmt.Sprintf("/B/%d", i))
		interestC := ndntestutil.MakeInterest(fmt.Sprintf("/C/%d", i))
		rxQueue.Rx(interestA, interestB, interestC)
		time.Sleep(50 * time.Microsecond)
	}
	time.Sleep(20 * time.Millisecond)

	e = server.Stop()
	assert.NoError(e)
	assert.Equal(100, nDataA0+nDataA1)
	assert.InDelta(60, nDataA0, 10)
	assert.InDelta(40, nDataA1, 10)
	assert.Equal(100, nNacksB)
}
