package pingserver_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping/pingtestenv"
	"github.com/usnistgov/ndn-dpdk/app/pingserver"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

func TestServer(t *testing.T) {
	assert, require := makeAR(t)

	face := pingtestenv.MakeMockFace()
	defer face.Close()
	face.DisableTxRecorders()

	nameA := ndn.ParseName("/A")
	nameB := ndn.ParseName("/B")
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
						Suffix:          ndn.ParseName("/Z"),
						FreshnessPeriod: 100,
						PayloadLen:      2000,
					},
				},
			},
			{
				Prefix: nameB,
				Replies: []pingserver.Reply{
					{
						Nack: an.NackCongestion,
					},
				},
			},
			{
				Prefix: ndn.ParseName("/C"),
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
	face.OnTxData(func(data *ndni.Data) {
		dataName := data.GetName()
		switch {
		case nameA.IsPrefixOf(dataName) && len(dataName) == 2:
			nDataA0++
		case nameA.IsPrefixOf(dataName) && len(dataName) == 3:
			nDataA1++
		default:
			assert.Fail("unexpected Data", "%s", data)
		}
	})
	face.OnTxNack(func(nack *ndni.Nack) {
		interestName := nack.GetInterest().GetName()
		switch {
		case nameB.IsPrefixOf(interestName) && len(interestName) == 2:
			nNacksB++
		default:
			assert.Fail("unexpected Nack", "%s", nack)
		}
	})

	server, e := pingserver.New(face, 0, cfg)
	require.NoError(e)
	defer server.Close()
	server.SetLCore(pingtestenv.SlaveLCores[0])

	server.Launch()

	rx := pingtestenv.MakeRxFunc(server.GetRxQueue())
	for i := 0; i < 100; i++ {
		interestA := makeInterest(fmt.Sprintf("/A/%d", i))
		interestB := makeInterest(fmt.Sprintf("/B/%d", i))
		interestC := makeInterest(fmt.Sprintf("/C/%d", i))
		rx(interestA, interestB, interestC)
		time.Sleep(50 * time.Microsecond)
	}
	time.Sleep(20 * time.Millisecond)

	e = server.Stop()
	assert.NoError(e)
	assert.Equal(100, nDataA0+nDataA1)
	assert.InDelta(60, nDataA0, 20)
	assert.InDelta(40, nDataA1, 20)
	assert.Equal(100, nNacksB)
}
