package main

import (
	"fmt"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/integ"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndnface"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()
	assert, require := integ.MakeAR(t)

	eal := dpdktestenv.InitEal()
	dpdktestenv.MakeDirectMp(4095, 0, 256)
	edp := dpdktestenv.NewEthDevPair(1, 1024, 64)

	faceA, e := ndnface.NewTxFace(edp.TxqA[0])
	require.NoError(e)
	defer faceA.Close()
	faceB := ndnface.NewRxFace(edp.RxqB[0])

	const RX_BURST_SIZE = 6

	nReceived := 0
	rxQuit := make(chan int)
	eal.Slaves[0].RemoteLaunch(func() int {
		pkts := make([]ndn.Packet, RX_BURST_SIZE)
		for {
			burstSize := faceB.RxBurst(pkts)
			nReceived += burstSize

			for _, pkt := range pkts[:burstSize] {
				if pkt.IsValid() {
					fmt.Printf("received l2=%v l3=%v len=%d\n", pkt.GetL2Type(), pkt.GetNetType(),
						pkt.Len())
				} else {
					fmt.Println("pkt invalid")
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
		pkts := make([]ndn.Packet, 3)
		pkts[0] = ndn.Packet{dpdktestenv.PacketFromHex("interest 050B name=0703080141 nonce=0A04CACBCCCD")}
		pkts[1] = ndn.Packet{dpdktestenv.PacketFromHex("data 0609 name=0703080141 meta=1400 content=1500")}
		pkts[2] = ndn.Packet{dpdktestenv.PacketFromHex("nack 6418 nack=FD032005(FD03210196~noroute) " +
			"payload=500D(interest 050B name=0703080141 nonce=0A04CACBCCCD)")}
		res := faceA.TxBurst(pkts)
		fmt.Printf("TxBurst res=%d\n", res)
		return 0
	})
	eal.Slaves[1].Wait()
	time.Sleep(time.Second)
	rxQuit <- 0

	fmt.Println(faceA.GetCounters())
	assert.Equal(3, nReceived)
}
