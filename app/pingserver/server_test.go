package pingserver_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/ping/pingtestenv"
	"github.com/usnistgov/ndn-dpdk/app/pingserver"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func TestServer(t *testing.T) {
	assert, require := makeAR(t)

	intFace := intface.MustNew()
	defer intFace.DFace.Close()

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

	server, e := pingserver.New(intFace.DFace, 0, cfg)
	require.NoError(e)
	defer server.Close()
	server.SetLCore(pingtestenv.SlaveLCores[0])
	pingtestenv.Demux3.GetInterestDemux().SetDest(0, server.GetRxQueue())

	nDataA0 := 0
	nDataA1 := 0
	nNacksB := 0
	go func() {
		for packet := range intFace.AFace.GetRx() {
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

	server.Launch()

	func() {
		tx := intFace.AFace.GetTx()
		for i := 0; i < 100; i++ {
			tx <- ndn.MakeInterest(fmt.Sprintf("/A/%d", i)).Packet
			tx <- ndn.MakeInterest(fmt.Sprintf("/B/%d", i)).Packet
			tx <- ndn.MakeInterest(fmt.Sprintf("/C/%d", i)).Packet
			time.Sleep(50 * time.Microsecond)
		}
		time.Sleep(80 * time.Millisecond)
		close(tx)
	}()

	e = server.Stop()
	assert.NoError(e)
	assert.Equal(100, nDataA0+nDataA1)
	assert.InDelta(60, nDataA0, 20)
	assert.InDelta(40, nDataA1, 20)
	assert.Equal(100, nNacksB)
}
