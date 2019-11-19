package pingserver_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"ndn-dpdk/app/ping/pingtestenv"
	"ndn-dpdk/app/pingserver"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestServer(t *testing.T) {
	assert, require := makeAR(t)

	face := pingtestenv.MakeMockFace()
	defer face.Close()
	face.DisableTxRecorders()

	cfg := pingserver.Config{
		Patterns: []pingserver.Pattern{
			{
				Prefix: ndn.MustParseName("/A"),
				Replies: []pingserver.Reply{
					{
						Weight:          60,
						FreshnessPeriod: 100 * time.Millisecond,
						PayloadLen:      1000,
					},
					{
						Weight:          40,
						Suffix:          ndn.MustParseName("/Z"),
						FreshnessPeriod: 100 * time.Millisecond,
						PayloadLen:      2000,
					},
				},
			},
			{
				Prefix: ndn.MustParseName("/B"),
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
		dataNameUri := dataName.String()
		switch {
		case strings.HasPrefix(dataNameUri, "/A") && dataName.Len() == 2:
			nDataA0++
		case strings.HasPrefix(dataNameUri, "/A") && dataName.Len() == 3 && strings.HasSuffix(dataNameUri, "/Z"):
			nDataA1++
		default:
			assert.Fail("unexpected Data", "%s", data)
		}
	})
	face.OnTxNack(func(nack *ndn.Nack) {
		interestName := nack.GetInterest().GetName()
		interestNameUri := interestName.String()
		switch {
		case strings.HasPrefix(interestNameUri, "/B") && interestName.Len() == 2:
			nNacksB++
		default:
			assert.Fail("unexpected Nack", "%s", nack)
		}
	})

	server, e := pingserver.New(face, cfg)
	require.NoError(e)
	defer server.Close()
	server.SetLCore(pingtestenv.SlaveLCores[0])

	rxQueue, e := dpdk.NewRing("PingServerRxQ", 1024, dpdk.NUMA_SOCKET_ANY, false, true)
	require.NoError(e)
	defer rxQueue.Close()
	server.SetRxQueue(rxQueue)

	server.Launch()

	for i := 0; i < 100; i++ {
		interestA := ndntestutil.MakeInterest(fmt.Sprintf("/A/%d", i))
		interestB := ndntestutil.MakeInterest(fmt.Sprintf("/B/%d", i))
		interestC := ndntestutil.MakeInterest(fmt.Sprintf("/C/%d", i))
		rxQueue.BurstEnqueue([]ndn.Packet{interestA.GetPacket(), interestB.GetPacket(), interestC.GetPacket()})
		time.Sleep(50 * time.Microsecond)
	}
	time.Sleep(20 * time.Millisecond)

	e = server.Stop()
	assert.NoError(e)
	assert.Equal(100, nDataA0+nDataA1)
	assert.InDelta(nDataA0, 60, 10)
	assert.InDelta(nDataA1, 40, 10)
	assert.Equal(100, nNacksB)
}
