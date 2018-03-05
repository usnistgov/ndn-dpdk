package ifacetestfixture

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

// Test fixture for sending and receiving packets between a pair of connected faces.
type Fixture struct {
	assert *assert.Assertions

	RxBurstSize   int        // RxBurst burst size
	TxLoops       int        // number of TX loops
	LossTolerance float64    // permitted packet loss
	RxLCore       dpdk.LCore // LCore for executing rxProc
	TxLCore       dpdk.LCore // LCore for txProc

	rxFace   IRxFace
	rxLooper iface.IRxLooper
	txFace   ITxFace

	NRxInterests int
	NRxData      int
	NRxNacks     int
}

func New(t *testing.T, rxFace IRxFace, rxLooper iface.IRxLooper,
	txFace ITxFace) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.assert = assert.New(t)

	fixture.RxBurstSize = 8
	fixture.TxLoops = 10000
	fixture.LossTolerance = 0.1

	eal := dpdktestenv.Eal
	fixture.RxLCore = eal.Slaves[0]
	fixture.TxLCore = eal.Slaves[1]

	fixture.rxFace = rxFace
	fixture.rxLooper = rxLooper
	fixture.txFace = txFace
	return fixture
}

func (fixture *Fixture) RunTest() {
	fixture.RxLCore.RemoteLaunch(fixture.rxProc)
	time.Sleep(200 * time.Millisecond)
	fixture.TxLCore.RemoteLaunch(fixture.txProc)

	fixture.TxLCore.Wait()
	time.Sleep(800 * time.Millisecond)
	fixture.rxLooper.StopRxLoop()
	fixture.RxLCore.Wait()
}

func (fixture *Fixture) rxProc() int {
	assert := fixture.assert

	cb, cbarg := iface.WrapRxCb(func(face iface.Face, burst iface.RxBurst) {
		check := func(l3pkt ndn.IL3Packet) {
			pkt := l3pkt.GetPacket().AsDpdkPacket()
			assert.Equal(fixture.rxFace.GetFaceId(), iface.FaceId(pkt.GetPort()))
			assert.NotZero(pkt.GetTimestamp())
			pkt.Close()
		}
		for _, interest := range burst.ListInterests() {
			fixture.NRxInterests++
			check(interest)
		}
		for _, data := range burst.ListData() {
			fixture.NRxData++
			check(data)
		}
		for _, nack := range burst.ListNacks() {
			fixture.NRxNacks++
			check(nack)
		}
	})
	fixture.rxLooper.RxLoop(fixture.RxBurstSize, cb, cbarg)
	return 0
}

func (fixture *Fixture) txProc() int {
	for i := 0; i < fixture.TxLoops; i++ {
		pkts := make([]ndn.Packet, 3)
		pkts[0] = ndntestutil.MakeInterest("050B name=0703080141 nonce=0A04CACBCCCD").GetPacket()
		pkts[1] = ndntestutil.MakeData("0609 name=0703080141 meta=1400 content=1500").GetPacket()
		pkts[2] = ndntestutil.MakeNack("6418 nack=FD032005(FD03210196~noroute) " +
			"payload=500D(interest 050B name=0703080141 nonce=0A04CACBCCCD)").GetPacket()
		fixture.txFace.TxBurst(pkts)
		time.Sleep(time.Millisecond)
	}
	return 0
}

func (fixture *Fixture) CheckCounters() {
	assert := fixture.assert

	txCnt := fixture.txFace.ReadCounters()
	assert.Equal(uint64(3*fixture.TxLoops), txCnt.TxL2.NFrames)
	assert.Equal(uint64(fixture.TxLoops), txCnt.TxL3.NInterests)
	assert.Equal(uint64(fixture.TxLoops), txCnt.TxL3.NData)
	assert.Equal(uint64(fixture.TxLoops), txCnt.TxL3.NNacks)

	rxCnt := fixture.rxFace.ReadCounters()
	assert.Equal(fixture.NRxInterests, int(rxCnt.RxL3.NInterests))
	assert.Equal(fixture.NRxData, int(rxCnt.RxL3.NData))
	assert.Equal(fixture.NRxNacks, int(rxCnt.RxL3.NNacks))
	assert.Equal(fixture.NRxInterests+fixture.NRxData+fixture.NRxNacks,
		int(rxCnt.RxL2.NFrames))

	assert.InEpsilon(fixture.TxLoops, fixture.NRxInterests, fixture.LossTolerance)
	assert.InEpsilon(fixture.TxLoops, fixture.NRxData, fixture.LossTolerance)
	assert.InEpsilon(fixture.TxLoops, fixture.NRxNacks, fixture.LossTolerance)
}
