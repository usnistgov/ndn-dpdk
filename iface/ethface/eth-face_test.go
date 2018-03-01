package ethface_test

import (
	"fmt"
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestEthFace(t *testing.T) {
	assert, require := dpdktestenv.MakeAR(t)

	eal := dpdktestenv.InitEal()
	dpdktestenv.MakeDirectMp(4095, ndn.SizeofPacketPriv(), 2000)
	mempools := iface.Mempools{
		IndirectMp: dpdktestenv.MakeIndirectMp(4095),
		NameMp:     dpdktestenv.MakeMp("name", 4095, 0, uint16(ndn.NAME_MAX_LENGTH)),
		// Normally headerMp does not need PrivRoom, but ring-based PMD would pass a 'header'
		// as first segment on the RX side, where PrivRoom is required.
		HeaderMp: dpdktestenv.MakeMp("header", 4095, ndn.SizeofPacketPriv(),
			ethface.SizeofHeaderMempoolDataRoom()),
	}
	edp := dpdktestenv.NewEthDevPair(1, 1024, 64)
	defer edp.Close()

	faceA, e := ethface.New(edp.PortA, mempools)
	require.NoError(e)
	defer faceA.Close()
	faceB, e := ethface.New(edp.PortB, mempools)
	require.NoError(e)
	defer faceB.Close()

	const RX_BURST_SIZE = 6
	const TX_LOOPS = 10000

	nReceived := 0
	eal.Slaves[0].RemoteLaunch(func() int {
		cb, cbarg := iface.WrapRxCb(func(face iface.Face, burst iface.RxBurst) {
			check := func(l3pkt ndn.IL3Packet) {
				nReceived++
				pkt := l3pkt.GetPacket().AsDpdkPacket()
				assert.Equal(faceB.GetFaceId(), iface.FaceId(pkt.GetPort()))
				assert.NotZero(pkt.GetTimestamp())
				pkt.Close()
			}
			for _, interest := range burst.ListInterests() {
				check(interest)
			}
			for _, data := range burst.ListData() {
				check(data)
			}
			for _, nack := range burst.ListNacks() {
				check(nack)
			}
		})
		faceB.RxLoop(RX_BURST_SIZE, cb, cbarg)
		return 0
	})

	eal.Slaves[1].RemoteLaunch(func() int {
		for i := 0; i < TX_LOOPS; i++ {
			pkts := make([]ndn.Packet, 3)
			pkts[0] = ndntestutil.MakePacket("interest 050B name=0703080141 nonce=0A04CACBCCCD")
			pkts[1] = ndntestutil.MakePacket("data 0609 name=0703080141 meta=1400 content=1500")
			pkts[2] = ndntestutil.MakePacket("nack 6418 nack=FD032005(FD03210196~noroute) " +
				"payload=500D(interest 050B name=0703080141 nonce=0A04CACBCCCD)")
			faceA.TxBurst(pkts)
			for _, pkt := range pkts {
				pkt.AsDpdkPacket().Close()
			}
			time.Sleep(time.Millisecond)
		}
		return 0
	})
	eal.Slaves[1].Wait()
	time.Sleep(time.Second)
	faceB.StopRxLoop()
	eal.Slaves[0].Wait()

	fmt.Println(edp.PortA.GetStats())
	fmt.Println(edp.PortB.GetStats())
	cntA := faceA.ReadCounters()
	fmt.Println(cntA)
	cntB := faceB.ReadCounters()
	fmt.Println(cntB)

	assert.EqualValues(3*TX_LOOPS, cntA.TxL2.NFrames)
	// TxL3 counters are unavailable because packets do not have L3PktType specified.

	assert.True(nReceived > TX_LOOPS*3*0.9)
	assert.EqualValues(nReceived, cntB.RxL2.NFrames)
}
