package main

import (
	"fmt"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/integ"
	"ndn-dpdk/ndn"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()
	assert, require := integ.MakeAR(t)

	eal := dpdktestenv.InitEal()
	dpdktestenv.MakeDirectMp(4095, ndn.SizeofPacketPriv(), 2000)
	indirectMp := dpdktestenv.MakeIndirectMp(4095)
	// Normally headerMp does not need PrivRoom, but ring-based PMD would pass a 'header' as first
	// segment on the RX side, where PrivRoom is required.
	headerMp := dpdktestenv.MakeMp("header", 4095, ndn.SizeofPacketPriv(),
		ethface.SizeofHeaderMempoolDataRoom())
	edp := dpdktestenv.NewEthDevPair(1, 1024, 64)

	faceA, e := ethface.New(edp.PortA, indirectMp, headerMp)
	require.NoError(e)
	defer faceA.Close()
	faceB, e := ethface.New(edp.PortB, indirectMp, headerMp)
	require.NoError(e)
	defer faceB.Close()

	const RX_BURST_SIZE = 6
	const TX_LOOPS = 10000

	nReceived := 0
	rxQuit := make(chan struct{})
	eal.Slaves[0].RemoteLaunch(func() int {
		pkts := make([]ndn.Packet, RX_BURST_SIZE)
		for {
			burstSize := faceB.RxBurst(pkts)
			nReceived += burstSize

			for _, pkt := range pkts[:burstSize] {
				if assert.True(pkt.IsValid()) {
					pkt.Close()
				}
			}

			select {
			case <-rxQuit:
				return 0
			default:
			}
		}
	})

	eal.Slaves[1].RemoteLaunch(func() int {
		for i := 0; i < TX_LOOPS; i++ {
			pkts := make([]ndn.Packet, 3)
			pkts[0] = ndn.Packet{dpdktestenv.PacketFromHex("interest 050B name=0703080141 nonce=0A04CACBCCCD")}
			pkts[1] = ndn.Packet{dpdktestenv.PacketFromHex("data 0609 name=0703080141 meta=1400 content=1500")}
			pkts[2] = ndn.Packet{dpdktestenv.PacketFromHex("nack 6418 nack=FD032005(FD03210196~noroute) " +
				"payload=500D(interest 050B name=0703080141 nonce=0A04CACBCCCD)")}
			faceA.TxBurst(pkts)
			pkts[0].Close()
			pkts[1].Close()
			pkts[2].Close()
			time.Sleep(time.Millisecond)
		}
		return 0
	})
	eal.Slaves[1].Wait()
	time.Sleep(time.Second)
	rxQuit <- struct{}{}

	fmt.Println(edp.PortA.GetStats())
	fmt.Println(edp.PortB.GetStats())
	cntA := faceA.ReadCounters()
	fmt.Println(cntA)
	cntB := faceB.ReadCounters()
	fmt.Println(cntB)

	assert.EqualValues(3*TX_LOOPS, cntA.TxL2.NFrames)
	// TxL3 counters are unavailable because packets do not have NdnPktType specified.

	assert.True(nReceived > TX_LOOPS*3*0.9)
	assert.EqualValues(nReceived, cntB.RxL2.NFrames)
}
