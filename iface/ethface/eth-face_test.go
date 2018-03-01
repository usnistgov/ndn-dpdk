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
		HeaderMp:   dpdktestenv.MakeMp("header", 4095, 0, ethface.SizeofHeaderMempoolDataRoom()),
	}
	evl := dpdktestenv.NewEthVLink(1024, 64, dpdktestenv.MPID_DIRECT)
	defer evl.Close()

	faceA, e := ethface.New(evl.PortA, mempools)
	require.NoError(e)
	defer faceA.Close()
	faceB, e := ethface.New(evl.PortB, mempools)
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
			pkts[0] = ndntestutil.MakeInterest("050B name=0703080141 nonce=0A04CACBCCCD").GetPacket()
			pkts[1] = ndntestutil.MakeData("0609 name=0703080141 meta=1400 content=1500").GetPacket()
			pkts[2] = ndntestutil.MakeNack("6418 nack=FD032005(FD03210196~noroute) " +
				"payload=500D(interest 050B name=0703080141 nonce=0A04CACBCCCD)").GetPacket()
			faceA.TxBurst(pkts)
			for _, pkt := range pkts {
				pkt.AsDpdkPacket().Close()
			}
			time.Sleep(time.Millisecond)
		}
		return 0
	})

	eal.Slaves[2].RemoteLaunch(evl.Bridge)

	eal.Slaves[1].Wait()
	time.Sleep(time.Second)
	faceB.StopRxLoop()
	eal.Slaves[0].Wait()

	fmt.Println(evl.PortA.GetStats())
	fmt.Println(evl.PortB.GetStats())
	cntA := faceA.ReadCounters()
	fmt.Println(cntA)
	cntB := faceB.ReadCounters()
	fmt.Println(cntB)

	assert.EqualValues(3*TX_LOOPS, cntA.TxL2.NFrames)
	assert.EqualValues(TX_LOOPS, cntA.TxL3.NInterests)
	assert.EqualValues(TX_LOOPS, cntA.TxL3.NData)
	assert.EqualValues(TX_LOOPS, cntA.TxL3.NNacks)

	assert.True(nReceived > TX_LOOPS*3*0.9)
	assert.EqualValues(nReceived, cntB.RxL2.NFrames)
}
