package ifacetestenv

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

// Test fixture for sending and receiving packets between a pair of connected faces.
type Fixture struct {
	t *testing.T

	PayloadLen    int       // Data payload length
	TxLoops       int       // number of TX loops
	LossTolerance float64   // permitted packet loss
	RxLCore       eal.LCore // LCore for executing RxLoop
	TxLCore       eal.LCore // LCore for executing TxLoop
	SendLCore     eal.LCore // LCore for executing sendProc

	rxFace    iface.Face
	rxDiscard map[iface.ID]iface.Face
	rxl       *iface.RxLoop
	txFace    iface.Face
	txl       *iface.TxLoop

	NRxInterests int
	NRxData      int
	NRxNacks     int
}

func New(t *testing.T, rxFace, txFace iface.Face) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.t = t

	fixture.TxLoops = 10000
	fixture.LossTolerance = 0.1

	slaves := eal.ListSlaveLCores()
	fixture.RxLCore = slaves[0]
	fixture.TxLCore = slaves[1]
	fixture.SendLCore = slaves[2]

	fixture.rxFace = rxFace
	fixture.rxDiscard = make(map[iface.ID]iface.Face)
	fixture.txFace = txFace

	CheckLocatorMarshal(t, rxFace.Locator())
	CheckLocatorMarshal(t, txFace.Locator())
	return fixture
}

func (fixture *Fixture) AddRxDiscard(face iface.Face) {
	fixture.rxDiscard[face.ID()] = face
}

func (fixture *Fixture) RunTest() {
	fixture.launchRx()
	fixture.txl = iface.NewTxLoop(fixture.txFace.NumaSocket())
	fixture.txl.SetLCore(fixture.TxLCore)
	fixture.txl.Launch()
	fixture.txl.AddFace(fixture.txFace)
	time.Sleep(200 * time.Millisecond)

	fixture.SendLCore.RemoteLaunch(fixture.sendProc)
	fixture.SendLCore.Wait()
	time.Sleep(800 * time.Millisecond)

	fixture.txl.Stop()
	fixture.rxl.Stop()
	fixture.txl.Close()
	fixture.rxl.Close()
}

func (fixture *Fixture) launchRx() {
	assert, require := testenv.MakeAR(fixture.t)

	fixture.rxl = iface.NewRxLoop(fixture.rxFace.NumaSocket())
	fixture.rxl.SetLCore(fixture.RxLCore)

	fixture.rxl.SetCallback(iface.WrapRxCb(func(burst iface.RxBurst) {
		check := func(l3pkt ndni.IL3Packet) (increment int) {
			pkt := l3pkt.GetPacket().AsMbuf()
			faceID := iface.ID(pkt.GetPort())
			if _, ok := fixture.rxDiscard[faceID]; !ok {
				assert.Equal(fixture.rxFace.ID(), faceID)
				assert.NotZero(pkt.GetTimestamp())
				increment = 1
			}
			pkt.Close()
			return increment
		}

		for _, interest := range burst.ListInterests() {
			fixture.NRxInterests += check(interest)
		}
		for _, data := range burst.ListData() {
			fixture.NRxData += check(data)
		}
		for _, nack := range burst.ListNacks() {
			fixture.NRxNacks += check(nack)
		}
	}))

	require.NoError(fixture.rxl.Launch())
	time.Sleep(50 * time.Millisecond)
	for _, rxg := range fixture.rxFace.ListRxGroups() {
		require.NoError(fixture.rxl.AddRxGroup(rxg))
	}
	for _, face := range fixture.rxDiscard {
		for _, rxg := range face.ListRxGroups() {
			fixture.rxl.AddRxGroup(rxg)
		}
	}
}

func (fixture *Fixture) sendProc() int {
	content := make([]byte, fixture.PayloadLen)
	for i := 0; i < fixture.TxLoops; i++ {
		pkts := make([]*ndni.Packet, 3)
		pkts[0] = ndnitestenv.MakeInterest("/A").GetPacket()
		pkts[1] = ndnitestenv.MakeData("/A", content).GetPacket()
		pkts[2] = ndni.MakeNackFromInterest(ndnitestenv.MakeInterest("/A"), an.NackNoRoute).GetPacket()
		fixture.txFace.TxBurst(pkts)
		time.Sleep(time.Millisecond)
	}
	return 0
}

func (fixture *Fixture) CheckCounters() {
	assert, _ := testenv.MakeAR(fixture.t)

	txCnt := fixture.txFace.ReadCounters()
	assert.Equal(3*fixture.TxLoops, int(txCnt.TxFrames))
	assert.Equal(fixture.TxLoops, int(txCnt.TxInterests))
	assert.Equal(fixture.TxLoops, int(txCnt.TxData))
	assert.Equal(fixture.TxLoops, int(txCnt.TxNacks))

	rxCnt := fixture.rxFace.ReadCounters()
	assert.Equal(fixture.NRxInterests, int(rxCnt.RxInterests))
	assert.Equal(fixture.NRxData, int(rxCnt.RxData))
	assert.Equal(fixture.NRxNacks, int(rxCnt.RxNacks))
	assert.Equal(fixture.NRxInterests+fixture.NRxData+fixture.NRxNacks,
		int(rxCnt.RxFrames))

	assert.InEpsilon(fixture.TxLoops, fixture.NRxInterests, fixture.LossTolerance)
	assert.InEpsilon(fixture.TxLoops, fixture.NRxData, fixture.LossTolerance)
	assert.InEpsilon(fixture.TxLoops, fixture.NRxNacks, fixture.LossTolerance)
}
