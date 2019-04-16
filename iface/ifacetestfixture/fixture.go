package ifacetestfixture

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

// Test fixture for sending and receiving packets between a pair of connected faces.
type Fixture struct {
	assert  *assert.Assertions
	require *require.Assertions

	TxLoops       int        // number of TX loops
	LossTolerance float64    // permitted packet loss
	RxLCore       dpdk.LCore // LCore for executing RxLoop
	TxLCore       dpdk.LCore // LCore for executing TxLoop
	SendLCore     dpdk.LCore // LCore for executing txProc

	rxFace    iface.IFace
	rxDiscard map[iface.FaceId]iface.IFace
	rxl       *iface.RxLoop
	txFace    iface.IFace
	txl       *iface.TxLoop

	NRxInterests int
	NRxData      int
	NRxNacks     int
}

func New(t *testing.T, rxFace, txFace iface.IFace) (fixture *Fixture) {
	fixture = new(Fixture)
	fixture.assert = assert.New(t)
	fixture.require = require.New(t)

	fixture.TxLoops = 10000
	fixture.LossTolerance = 0.1

	slaves := dpdk.ListSlaveLCores()
	fixture.RxLCore = slaves[0]
	fixture.TxLCore = slaves[1]
	fixture.SendLCore = slaves[2]

	fixture.rxFace = rxFace
	fixture.rxDiscard = make(map[iface.FaceId]iface.IFace)
	fixture.txFace = txFace
	return fixture
}

func (fixture *Fixture) AddRxDiscard(face iface.IFace) {
	fixture.rxDiscard[face.GetFaceId()] = face
}

func (fixture *Fixture) RunTest() {
	fixture.launchRx()
	fixture.txl = iface.NewTxLoop(fixture.txFace)
	fixture.txl.SetLCore(fixture.TxLCore)
	fixture.txl.Launch()
	time.Sleep(200 * time.Millisecond)

	fixture.SendLCore.RemoteLaunch(fixture.txProc)
	fixture.SendLCore.Wait()
	time.Sleep(800 * time.Millisecond)

	fixture.txl.Stop()
	fixture.rxl.Stop()
	fixture.txl.Close()
	fixture.rxl.Close()
}

func (fixture *Fixture) launchRx() {
	assert, require := fixture.assert, fixture.require

	fixture.rxl = iface.NewRxLoop(fixture.rxFace.GetNumaSocket())
	fixture.rxl.SetLCore(fixture.RxLCore)

	cb, cbarg := iface.WrapRxCb(func(burst iface.RxBurst) {
		check := func(l3pkt ndn.IL3Packet) {
			pkt := l3pkt.GetPacket().AsDpdkPacket()
			faceId := iface.FaceId(pkt.GetPort())
			if _, ok := fixture.rxDiscard[faceId]; !ok {
				assert.Equal(fixture.rxFace.GetFaceId(), faceId)
				assert.NotZero(pkt.GetTimestamp())
			}
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
	fixture.rxl.SetCallback(cb, cbarg)

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

func (fixture *Fixture) txProc() int {
	for i := 0; i < fixture.TxLoops; i++ {
		pkts := make([]ndn.Packet, 3)
		pkts[0] = ndntestutil.MakeInterest("/A").GetPacket()
		pkts[1] = ndntestutil.MakeData("/A").GetPacket()
		pkts[2] = ndn.MakeNackFromInterest(ndntestutil.MakeInterest("/A"), ndn.NackReason_NoRoute).GetPacket()
		fixture.txFace.TxBurst(pkts)
		time.Sleep(time.Millisecond)
	}
	return 0
}

func (fixture *Fixture) CheckCounters() {
	assert := fixture.assert

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
